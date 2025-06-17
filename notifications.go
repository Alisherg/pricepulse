package main

import (
	"fmt"
	"log"
	"os"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func sendEmailNotification(toEmail, subject, assetID string, priceChange float64, newPrice float64) {
	apiKey := os.Getenv("SENDGRID_API_KEY")
	if apiKey == "" {
		log.Println("SENDGRID_API_KEY not set. Skipping email notification.")
		return
	}

	from := mail.NewEmail("PricePulse", os.Getenv("SENDGRID_FROM_EMAIL"))
	to := mail.NewEmail("Valued User", toEmail)

	plainTextContent := fmt.Sprintf(
		"Alert for %s! It moved by %.2f%%. The new price is $%.2f.",
		assetID, priceChange, newPrice,
	)
	htmlContent := fmt.Sprintf(
		"<strong>Alert for %s!</strong> It moved by <strong>%.2f%%</strong>. The new price is <strong>$%.2f</strong>.",
		assetID, priceChange, newPrice,
	)
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(apiKey)
	response, err := client.Send(message)
	if err != nil {
		log.Printf("Failed to send email: %v", err)
	} else if response.StatusCode >= 400 {
		log.Printf("SendGrid returned an error: %d - %s", response.StatusCode, response.Body)
	} else {
		log.Printf("Email sent successfully to %s!", toEmail)
	}
}
