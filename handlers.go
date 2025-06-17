package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// The structure for the Signal entity
type Signal struct {
	UserID                    string    `firestore:"userId"`
	Email                     string    `firestore:"email"`
	AssetID                   string    `firestore:"assetId"`
	ChangeThresholdPercentage float64   `firestore:"changeThresholdPercentage"`
	PriceAtCreation           float64   `firestore:"priceAtCreation"`
	Status                    string    `firestore:"status"`
	CreatedAt                 time.Time `firestore:"createdAt"`
}

// The struct to hold analysis results
type AnalysisResult struct {
	AssetId             string
	TimeWindowHours     int
	SimpleMovingAverage float64
	DataPointsUsed      int
}

// rootHandler serves the main index.html page.
func (a *App) rootHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Could not parse template", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// healthCheckHandler reports the status of the application.
func (a *App) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// createUserHandler handles user creation requests.
func (a *App) createUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}
	var data struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, _, err := a.db.Collection("users").Add(context.Background(), map[string]string{"username": data.Username})
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "user created"})
}

// createSignalHandler handles the creation of a new signal via API.
func (a *App) createSignalHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}
	var signal Signal
	if err := json.NewDecoder(r.Body).Decode(&signal); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	priceData, err := a.priceFetcher(signal.AssetID, "https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd")
	if err != nil {
		http.Error(w, "Failed to fetch current price for signal creation", http.StatusInternalServerError)
		return
	}
	currentPrice, ok := priceData[signal.AssetID]["usd"].(float64)
	if !ok {
		http.Error(w, "Failed to parse current price", http.StatusInternalServerError)
		return
	}
	signal.PriceAtCreation = currentPrice
	signal.Status = "active"
	signal.CreatedAt = time.Now()
	_, _, err = a.db.Collection("signals").Add(context.Background(), signal)
	if err != nil {
		http.Error(w, "Failed to create signal", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "signal created"})
}

// collectDataHandler fetches the current price of an asset and checks active signals.
func (a *App) collectDataHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	assetID := "bitcoin"
	priceData, err := a.priceFetcher(assetID, "https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd")
	if err != nil {
		http.Error(w, "Failed to fetch price data", http.StatusInternalServerError)
		return
	}
	currentPrice, _ := priceData[assetID]["usd"].(float64)
	a.db.Collection("price_history").Add(ctx, map[string]interface{}{"assetId": assetID, "price": currentPrice, "timestamp": time.Now()})
	iter := a.db.Collection("signals").Where("assetId", "==", assetID).Where("status", "==", "active").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err != nil {
			break
		}
		var s Signal
		doc.DataTo(&s)
		priceChange := ((currentPrice - s.PriceAtCreation) / s.PriceAtCreation) * 100
		absPriceChange := math.Abs(priceChange)
		log.Printf("Checking signal for user %s. Asset: %s. Current Change: %.2f%%. Threshold: %.2f%%", s.UserID, s.AssetID, absPriceChange, s.ChangeThresholdPercentage)
		if absPriceChange >= s.ChangeThresholdPercentage {
			log.Printf("!!! SIGNAL TRIGGERED for user %s! Price moved by %.2f%% !!!", s.UserID, priceChange)
			subject := fmt.Sprintf("Price Alert for %s", s.AssetID)
			sendEmailNotification(s.Email, subject, s.AssetID, priceChange, currentPrice)
			_, err := doc.Ref.Update(ctx, []firestore.Update{{Path: "status", Value: "triggered"}})
			if err != nil {
				log.Printf("Failed to update signal status: %v", err)
			}
		}
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "data collected and signals checked", "price": currentPrice})
}

// analysisHandler calculates and returns a simple analysis of the price data.
func (a *App) analysisHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	assetID := "bitcoin"
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)
	iter := a.db.Collection("price_history").Where("assetId", "==", assetID).Where("timestamp", ">=", twentyFourHoursAgo).Documents(ctx)
	defer iter.Stop()
	var totalPrice float64
	var count int
	for {
		doc, err := iter.Next()
		if err != nil {
			break
		}
		priceData := doc.Data()
		price, ok := priceData["price"].(float64)
		if !ok {
			continue
		}
		totalPrice += price
		count++
	}
	if count == 0 {
		http.Error(w, "Not enough data for analysis", http.StatusNotFound)
		return
	}
	averagePrice := totalPrice / float64(count)
	response := map[string]interface{}{"assetId": assetID, "time_window_hours": 24, "simple_moving_average": averagePrice, "data_points_used": count}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// showNewSignalFormHandler renders the form to create a new signal.
func (a *App) showNewSignalFormHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/new_signal_form.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// handleCreateSignalForm processes the form submission from the UI.
func (a *App) handleCreateSignalForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Could not parse form", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	assetID := r.FormValue("assetId")
	threshold, _ := strconv.ParseFloat(r.FormValue("threshold"), 64)

	// Fetch current price
	priceData, err := a.priceFetcher(assetID, "https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd")
	if err != nil {
		http.Error(w, "Could not fetch current price", http.StatusInternalServerError)
		return
	}
	currentPrice, ok := priceData[assetID]["usd"].(float64)
	if !ok {
		http.Error(w, "Could not parse current price", http.StatusInternalServerError)
		return
	}

	// Save signal to Firestore
	signal := Signal{
		UserID:                    email,
		Email:                     email,
		AssetID:                   assetID,
		ChangeThresholdPercentage: threshold,
		PriceAtCreation:           currentPrice,
		Status:                    "active",
		CreatedAt:                 time.Now(),
	}
	_, _, err = a.db.Collection("signals").Add(context.Background(), signal)
	if err != nil {
		http.Error(w, "Could not save signal to database", http.StatusInternalServerError)
		return
	}

	// Redirect user to their signals page after creation
	http.Redirect(w, r, "/signals/"+email, http.StatusSeeOther)
}

// viewUserSignalsHandler fetches and displays all signals for a given user.
func (a *App) viewUserSignalsHandler(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimPrefix(r.URL.Path, "/signals/")
	if email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}
	ctx := context.Background()

	var activeSignals []Signal
	iterSignals := a.db.Collection("signals").Where("email", "==", email).Where("status", "==", "active").Documents(ctx)
	for {
		doc, err := iterSignals.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			http.Error(w, "Failed to retrieve signals", http.StatusInternalServerError)
			return
		}
		var s Signal
		doc.DataTo(&s)
		activeSignals = append(activeSignals, s)
	}
	iterSignals.Stop()

	assetID := "bitcoin"
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)
	iterAnalysis := a.db.Collection("price_history").Where("assetId", "==", assetID).Where("timestamp", ">=", twentyFourHoursAgo).Documents(ctx)
	var totalPrice float64
	var count int
	for {
		doc, err := iterAnalysis.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			http.Error(w, "Failed to retrieve price history for analysis", http.StatusInternalServerError)
			return
		}
		priceData := doc.Data()
		price, ok := priceData["price"].(float64)
		if !ok {
			continue
		}
		totalPrice += price
		count++
	}
	iterAnalysis.Stop()

	analysisData := AnalysisResult{}
	if count > 0 {
		analysisData = AnalysisResult{
			AssetId:             assetID,
			TimeWindowHours:     24,
			SimpleMovingAverage: totalPrice / float64(count),
			DataPointsUsed:      count,
		}
	}

	// Combine all data for the template
	pageData := map[string]interface{}{
		"Email":         email,
		"ActiveSignals": activeSignals,
		"Analysis":      analysisData,
	}

	tmpl, err := template.ParseFiles("templates/user_page.html")
	if err != nil {
		http.Error(w, "Could not parse user page template", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, pageData)
}
