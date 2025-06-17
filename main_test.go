package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// clearCollection deletes all documents in a Firestore collection.
func clearCollection(ctx context.Context, client *firestore.Client, collName string) error {
	iter := client.Collection(collName).Documents(ctx)
	numDeleted := 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		_, err = doc.Ref.Delete(ctx)
		if err != nil {
			return err
		}
		numDeleted++
	}
	return nil
}

// Unit Test for getPriceFromCoinGecko
func TestGetPriceFromCoinGecko(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"bitcoin":{"usd":65000.50}}`)
	}))
	defer server.Close()

	priceData, err := getPriceFromCoinGecko("bitcoin", server.URL+"?ids=%s&vs_currencies=usd")
	if err != nil {
		t.Fatalf("getPriceFromCoinGecko failed: %v", err)
	}

	price, ok := priceData["bitcoin"]["usd"].(float64)
	if !ok {
		t.Fatal("Could not parse price from response")
	}

	if price != 65000.50 {
		t.Errorf("expected price 65000.50, got %f", price)
	}
}

// Integration Test for collectDataHandler
func TestCollectDataHandler(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("Skipping integration test: FIRESTORE_EMULATOR_HOST not set.")
	}
	ctx := context.Background()
	projectID := "testing-project"
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("Failed to create Firestore client for emulator: %v", err)
	}
	defer client.Close()

	clearCollection(ctx, client, "signals")
	clearCollection(ctx, client, "price_history")

	// Create our App instance for testing.
	app := &App{
		db: client,
		// Inject a FAKE priceFetcher function
		priceFetcher: func(assetID string, apiURL string) (map[string]map[string]interface{}, error) {
			return map[string]map[string]interface{}{
				"bitcoin": {
					"usd": 68000.00,
				},
			}, nil
		},
	}

	signal := Signal{
		UserID:                    "test-user",
		AssetID:                   "bitcoin",
		ChangeThresholdPercentage: 2.0,
		PriceAtCreation:           66000.00,
		Status:                    "active",
		CreatedAt:                 time.Now(),
	}
	docRef, _, err := client.Collection("signals").Add(ctx, signal)
	if err != nil {
		t.Fatalf("Failed to add test signal: %v", err)
	}

	// Simulate a request to collect data
	req := httptest.NewRequest("GET", "/collect-data", nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(app.collectDataHandler)
	handler.ServeHTTP(rr, req)

	// Check the signal in Firestore to see if it was triggered.
	doc, err := docRef.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get signal document from Firestore: %v", err)
	}
	var updatedSignal Signal
	doc.DataTo(&updatedSignal)

	if updatedSignal.Status != "triggered" {
		t.Errorf("expected signal status to be 'triggered', but got '%s'", updatedSignal.Status)
	}
}

// Integration Test for the /analysis endpoint
func TestAnalysisHandler(t *testing.T) {
	// Skip test if emulator is not running
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("Skipping integration test: FIRESTORE_EMULATOR_HOST not set.")
	}

	ctx := context.Background()
	projectID := "testing-project"
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("Failed to create Firestore client for emulator: %v", err)
	}
	defer client.Close()

	if err := clearCollection(ctx, client, "price_history"); err != nil {
		t.Fatalf("Failed to clear collection: %v", err)
	}

	// Set up test data in Firestore
	priceCollection := client.Collection("price_history")

	// Add 3 data points within the last 24 hours. The handler SHOULD average these.
	_, _, _ = priceCollection.Add(ctx, map[string]interface{}{"assetId": "bitcoin", "price": 60000.0, "timestamp": time.Now().Add(-1 * time.Hour)})
	_, _, _ = priceCollection.Add(ctx, map[string]interface{}{"assetId": "bitcoin", "price": 61000.0, "timestamp": time.Now().Add(-2 * time.Hour)})
	_, _, _ = priceCollection.Add(ctx, map[string]interface{}{"assetId": "bitcoin", "price": 62000.0, "timestamp": time.Now().Add(-3 * time.Hour)})

	_, _, _ = priceCollection.Add(ctx, map[string]interface{}{"assetId": "bitcoin", "price": 50000.0, "timestamp": time.Now().Add(-30 * time.Hour)})

	app := &App{db: client}
	handler := http.HandlerFunc(app.analysisHandler)

	req := httptest.NewRequest("GET", "/analysis", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Could not decode JSON response: %v", err)
	}

	if dataPoints, ok := response["data_points_used"].(float64); !ok || dataPoints != 3 {
		t.Errorf("expected 'data_points_used' to be 3, got %v", response["data_points_used"])
	}

	expectedAverage := 61000.0
	if avg, ok := response["simple_moving_average"].(float64); !ok || avg != expectedAverage {
		t.Errorf("expected 'simple_moving_average' to be %f, got %v", expectedAverage, response["simple_moving_average"])
	}
}
