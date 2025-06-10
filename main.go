package main

import (
	"html/template"
	"log"
	"net/http"
)

// PageData is a struct that holds the data we want to pass to the HTML template.
type PageData struct {
	UserInput string
}

// handler for the root page "/"
func rootHandler(w http.ResponseWriter, r *http.Request) {
	// Parse our HTML file.
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create an empty PageData object
	data := PageData{UserInput: ""}

	// Execute the template, writing the generated HTML to the response.
	tmpl.Execute(w, data)
}

// handler for the "/echo" page which processes the form submission
func echoHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get the text from the form field named "userInput".
	userInput := r.FormValue("userInput")

	// Parse the HTML file again.
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a PageData object containing the user's submitted text.
	data := PageData{UserInput: userInput}

	// Execute the template with the user's data.
	tmpl.Execute(w, data)
}

func main() {
	// Register the rootHandler function to handle requests to the root URL "/".
	http.HandleFunc("/", rootHandler)

	// Register the echoHandler function to handle requests to the "/echo" URL.
	http.HandleFunc("/echo", echoHandler)

	log.Println("Server starting on port 8080...")
	// Start the HTTP server on port 8080.
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
