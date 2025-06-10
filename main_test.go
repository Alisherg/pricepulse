package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// This test is designed to ensure that the formatting logic works correctly.
func TestFormatCurrency(t *testing.T) {
	testCases := []struct {
		name     string
		input    float64
		expected string
	}{
		{"Standard Case", 1234.56, "$1,234.56"},
		{"Zero Case", 0.0, "$0.00"},
		{"No Cents Case", 500.0, "$500.00"},
		{"Large Number Case", 9876543.21, "$9,876,543.21"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := formatCurrency(tc.input)
			if actual != tc.expected {
				t.Errorf("expected %q, but got %q", tc.expected, actual)
			}
		})
	}
}

// TestRootHandler tests the rootHandler function to ensure it serves the main page correctly.
func TestRootHandler(t *testing.T) {
	// Create a mock HTTP request.
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to capture the response.
	rr := httptest.NewRecorder()

	// Create a handler from our rootHandler function and serve the request.
	handler := http.HandlerFunc(rootHandler)
	handler.ServeHTTP(rr, req)

	// Check the status code.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body for the expected formatted currency string.
	expected := "$12,345.67"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: body does not contain %q", expected)
	}
}
