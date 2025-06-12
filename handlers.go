package main

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"math"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
)

// the structure for the Signal entity
type Signal struct {
	UserID                    string    `firestore:"userId"`
	AssetID                   string    `firestore:"assetId"`
	ChangeThresholdPercentage float64   `firestore:"changeThresholdPercentage"`
	PriceAtCreation           float64   `firestore:"priceAtCreation"`
	Status                    string    `firestore:"status"`
	CreatedAt                 time.Time `firestore:"createdAt"`
}

// rootHandler serves the main index.html page.
func (a *App) rootHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.html")
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

// createSignalHandler handles the creation of a new signal.
func (a *App) createSignalHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}
	var reqBody struct {
		UserID                    string  `json:"userId"`
		AssetID                   string  `json:"assetId"`
		ChangeThresholdPercentage float64 `json:"changeThresholdPercentage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	priceData, err := a.priceFetcher(reqBody.AssetID, "https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd")
	if err != nil {
		http.Error(w, "Failed to fetch current price for signal creation", http.StatusInternalServerError)
		return
	}
	currentPrice, ok := priceData[reqBody.AssetID]["usd"].(float64)
	if !ok {
		http.Error(w, "Failed to parse current price", http.StatusInternalServerError)
		return
	}

	signal := Signal{
		UserID:                    reqBody.UserID,
		AssetID:                   reqBody.AssetID,
		ChangeThresholdPercentage: reqBody.ChangeThresholdPercentage,
		PriceAtCreation:           currentPrice,
		Status:                    "active",
		CreatedAt:                 time.Now(),
	}

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

	a.db.Collection("price_history").Add(ctx, map[string]interface{}{
		"assetId":   assetID,
		"price":     currentPrice,
		"timestamp": time.Now(),
	})

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
			_, err := doc.Ref.Update(ctx, []firestore.Update{
				{Path: "status", Value: "triggered"},
			})
			if err != nil {
				log.Printf("Failed to update signal status: %v", err)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "data collected and signals checked", "price": currentPrice})
}
