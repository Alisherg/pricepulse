package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
)

// priceFetcherFunc defines a function type for fetching prices.
type priceFetcherFunc func(assetID string, apiURL string) (map[string]map[string]interface{}, error)

// App struct holds the Firestore client and the price fetching function.
type App struct {
	db           *firestore.Client
	priceFetcher priceFetcherFunc
}

func main() {
	ctx := context.Background()
	var client *firestore.Client
	var err error

	// Check for a "production" environment flag
	if os.Getenv("ENV") == "production" {
		log.Println("Running in PRODUCTION mode. Connecting to live Firestore.")

		projectID := os.Getenv("GCP_PROJECT")
		databaseID := os.Getenv("FIRESTORE_DATABASE_ID")

		client, err = firestore.NewClientWithDatabase(ctx, projectID, databaseID)

	} else {
		log.Println("Running in LOCAL mode. Connecting to Firestore emulator.")

		if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
			log.Fatal("You are running in local mode but FIRESTORE_EMULATOR_HOST is not set.")
		}

		projectID := "pricepulse-demo"
		client, err = firestore.NewClient(ctx, projectID)
	}

	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	// Create a new App instance, "injecting" the REAL priceFetcher function.
	app := &App{
		db:           client,
		priceFetcher: getPriceFromCoinGecko,
	}

	http.HandleFunc("/", app.rootHandler)
	http.HandleFunc("/health", app.healthCheckHandler)
	http.HandleFunc("/users", app.createUserHandler)
	http.HandleFunc("/signals", app.createSignalHandler)
	http.HandleFunc("/collect-data", app.collectDataHandler)
	http.HandleFunc("/analysis", app.analysisHandler)
	http.HandleFunc("/new-signal", app.showNewSignalFormHandler)
	http.HandleFunc("/create-signal", app.handleCreateSignalForm)
	http.HandleFunc("/signals/", app.viewUserSignalsHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
