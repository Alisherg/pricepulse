package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// getPriceFromCoinGecko fetches the price of an asset from CoinGecko.
func getPriceFromCoinGecko(assetID string, apiURL string) (map[string]map[string]interface{}, error) {
	url := fmt.Sprintf(apiURL, assetID)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data map[string]map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}
