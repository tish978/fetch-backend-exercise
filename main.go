package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// In-memory storage for receipts and their calculated points
var (
	receipts = make(map[string]Receipt)
	points   = make(map[string]int)
	mu       sync.Mutex // Mutex to handle concurrent access to the maps
)

// Item represents a single item in the receipt
type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

// Receipt represents a single receipt's structure
type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []Item `json:"items"`
	Total        string `json:"total"`
}

// ReceiptPoints represents the points awarded from a receipt
type ReceiptPoints struct {
	Points int `json:"points"`
}

// main function to start the HTTP server and sets up the endpoints
func main() {
	fmt.Println("Starting server on port 8081...")
	http.HandleFunc("/test", testHandler)
	http.HandleFunc("/receipts/process", processReceiptsHandler)
	http.HandleFunc("/receipts/", getPointsHandler)
	http.ListenAndServe(":8081", nil)
}

// testHandler is a simple endpoint to verify that the server is running
func testHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Test handler is working!")
}

// processReceiptsHandler processes a receipt and calculates its points
func processReceiptsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Processing receipt...")
	var receipt Receipt

	// Decode the JSON request body into a Receipt struct
	if err := json.NewDecoder(r.Body).Decode(&receipt); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate receipt fields
	if receipt.Retailer == "" || receipt.PurchaseDate == "" || receipt.PurchaseTime == "" ||
		receipt.Total == "" || len(receipt.Items) == 0 {
		http.Error(w, "Invalid receipt: missing required fields", http.StatusBadRequest)
		return
	}

	// Generate a unique ID for the receipt and calculate points for it
	id := uuid.New().String()
	mu.Lock()
	receipts[id] = receipt
	points[id] = calculatePoints(receipt)
	mu.Unlock()

	response := map[string]string{"id": id}

	// Encode the response as JSON
	json.NewEncoder(w).Encode(response)
}

// getPointsHandler retrieves the points for a given receipt
func getPointsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Getting points for receipt...")

	// Extract the receipt ID from the URL
	id := r.URL.Path[len("/receipts/") : len(r.URL.Path)-len("/points")]

	mu.Lock()
	pointsValue, ok := points[id]
	mu.Unlock()

	if !ok {
		http.Error(w, "receipt not found", http.StatusNotFound)
		return
	}

	response := ReceiptPoints{Points: pointsValue}
	json.NewEncoder(w).Encode(response)
}

// calculatePoints calculates the points for a given receipt based on specific rules
func calculatePoints(receipt Receipt) int {
	points := 0

	// Rule 1: One point for every alphanumeric character in the retailer name.
	alphanumeric := regexp.MustCompile("[a-zA-Z0-9]")
	points += len(alphanumeric.FindAllString(receipt.Retailer, -1))
	fmt.Printf("Rule 1: points = %d\n", points)

	// Rule 2: 50 points if the total is a round dollar amount with no cents.
	if total, err := strconv.ParseFloat(receipt.Total, 64); err == nil {
		if math.Mod(total, 1.0) == 0 {
			points += 50
			fmt.Printf("Rule 2: points = %d\n", points)
		}
	}

	// Rule 3: 25 points if the total is a multiple of 0.25.
	if total, err := strconv.ParseFloat(receipt.Total, 64); err == nil {
		if math.Mod(total, 0.25) == 0 {
			points += 25
			fmt.Printf("Rule 3: points = %d\n", points)
		}
	}

	// Rule 4: 5 points for every two items on the receipt.
	points += (len(receipt.Items) / 2) * 5
	fmt.Printf("Rule 4: points = %d\n", points)

	// Rule 5: Points for item description length.
	for _, item := range receipt.Items {

		originalLength := len(item.ShortDescription)
		trimmed := strings.TrimSpace(item.ShortDescription)
		alphanumeric := regexp.MustCompile("[a-zA-Z0-9]")
		cleaned := alphanumeric.FindAllString(trimmed, -1)
		cleanedLength := len(strings.Join(cleaned, ""))
		fmt.Printf("Item: %s, Original Length: %d, Trimmed: %s, Cleaned Length: %d\n",
			item.ShortDescription, originalLength, trimmed, cleanedLength)

		if cleanedLength%3 == 0 {
			if price, err := strconv.ParseFloat(item.Price, 64); err == nil {
				additionalPoints := int(math.Ceil(price * 0.2))
				fmt.Printf("Price: %f, Additional Points: %d\n", price, additionalPoints)
				points += additionalPoints
				fmt.Printf("Rule 5: points = %d\n", points)
			}
		}
	}

	// Rule 6: 6 points if the day in the purchase date is odd.
	if date, err := time.Parse("2006-01-02", receipt.PurchaseDate); err == nil {
		if date.Day()%2 != 0 {
			points += 6
			fmt.Printf("Rule 6: points = %d\n", points)
		}
	}

	// Rule 7: 10 points if the time of purchase is after 2:00pm and before 4:00pm.
	if purchaseTime, err := time.Parse("15:04", receipt.PurchaseTime); err == nil {
		if purchaseTime.Hour() >= 14 && purchaseTime.Hour() < 16 {
			points += 10
			fmt.Printf("Rule 7: points = %d\n", points)
		}
	}

	return points
}
