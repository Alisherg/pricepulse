package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
)

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
