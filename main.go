package main

import (
	"html/template"
	"log"
	"net/http"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// formatCurrency takes a float64 and returns a USD-formatted string.
// e.g., 12345.67 -> "$12,345.67"
func formatCurrency(n float64) string {
	// Using a message printer for proper comma formatting.
	p := message.NewPrinter(language.English)
	formatted := p.Sprintf("$%.2f", n)
	log.Printf("Formatted currency: %s", formatted)
	return formatted
}

// rootHandler serves the main page and formats a price as currency.
func rootHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Example usage of formatCurrency
	price := 12345.67
	formattedPrice := formatCurrency(price)

	data := map[string]string{
		"Price": formattedPrice,
	}

	tmpl.Execute(w, data)
}

func main() {
	http.HandleFunc("/", rootHandler)

	log.Println("Server starting on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
