package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// Verifies points are calculated correctly for a sample receipt
func TestCalculatePoints(t *testing.T) {
	receipt := Receipt{
		Retailer:     "Target",
		PurchaseDate: "2022-01-01",
		PurchaseTime: "13:01",
		Items: []Item{
			{ShortDescription: "Mountain Dew 12PK", Price: "6.49"},
			{ShortDescription: "Emils Cheese Pizza", Price: "12.25"},
			{ShortDescription: "Knorr Creamy Chicken", Price: "1.26"},
			{ShortDescription: "Doritos Nacho Cheese", Price: "3.35"},
			{ShortDescription: "   Klarbrunn 12-PK 12 FL OZ  ", Price: "12.00"},
		},
		Total: "35.35",
	}

	expectedPoints := 6 + 10 + 4 + 6 // 6 (Target) + 10 (5 points for every 2 items) + 4 (trimmed length) + 6 (odd day)
	points := calculatePoints(receipt)

	if points != expectedPoints {
		t.Errorf("Expected %d points; got %d", expectedPoints, points)
	}
}

// Verifies that 'receipts/process/' processes a receipt and returns an ID
func TestProcessReceiptsHandler(t *testing.T) {
	receipt := Receipt{
		Retailer:     "Target",
		PurchaseDate: "2022-01-01",
		PurchaseTime: "13:01",
		Items: []Item{
			{ShortDescription: "Mountain Dew 12PK", Price: "6.49"},
			{ShortDescription: "Emils Cheese Pizza", Price: "12.25"},
			{ShortDescription: "Knorr Creamy Chicken", Price: "1.26"},
			{ShortDescription: "Doritos Nacho Cheese", Price: "3.35"},
			{ShortDescription: "   Klarbrunn 12-PK 12 FL OZ  ", Price: "12.00"},
		},
		Total: "35.35",
	}

	body, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("Failed to marshal receipt: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/receipts/process", bytes.NewReader(body))
	w := httptest.NewRecorder()
	processReceiptsHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK; got %v", resp.Status)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {

		t.Fatalf("Failed to decode response: %v",
			err)
	}

	if _, ok := result["id"]; !ok {
		t.Fatalf("Expected id in response; got %v", result)
	}
}

// Verifies that processing an empty receipt returns a 400 Bad Request
func TestProcessEmptyReceipt(t *testing.T) {
	receipt := Receipt{}

	body, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("Failed to marshal receipt: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/receipts/process", bytes.NewReader(body))
	w := httptest.NewRecorder()
	processReceiptsHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status BadRequest; got %v", resp.Status)
	}
}

// Verifies that processing a receipt with missing fields returns a 400 Bad Request
func TestProcessReceiptMissingFields(t *testing.T) {
	receipt := Receipt{
		Retailer: "Target",
	}

	body, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("Failed to marshal receipt: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/receipts/process", bytes.NewReader(body))
	w := httptest.NewRecorder()
	processReceiptsHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status BadRequest; got %v", resp.Status)
	}
}

// Verifies that an invalid JSON payload returns a 400 Bad Request
func TestProcessInvalidJSON(t *testing.T) {
	body := []byte("{invalid_json}")

	req := httptest.NewRequest(http.MethodPost, "/receipts/process", bytes.NewReader(body))
	w := httptest.NewRecorder()
	processReceiptsHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status BadRequest; got %v", resp.Status)
	}
}

// Verifies that points are calculated correctly for a retailer with special characters
func TestCalculatePointsSpecialCharacters(t *testing.T) {
	receipt := Receipt{
		Retailer: "T@rget!123",
		Items: []Item{
			{ShortDescription: "Mountain Dew 12PK", Price: "6.49"},
		},
		Total: "6.49",
	}

	expectedPoints := 8 + 2 // 8 (Retailer name) + 2 (trimmed length)
	points := calculatePoints(receipt)

	if points != expectedPoints {
		t.Errorf("Expected %d points; got %d", expectedPoints, points)
	}
}

// Verifies that querying a non-existent receipt returns a 404 Not Found
func TestGetPointsNonExistentReceipt(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/receipts/nonexistent/points", nil)
	w := httptest.NewRecorder()
	getPointsHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status NotFound; got %v", resp.Status)
	}
}

// Verifies that concurrent requests can be handled
func TestConcurrentRequests(t *testing.T) {
	receipt := Receipt{
		Retailer:     "Target",
		PurchaseDate: "2022-01-01",
		PurchaseTime: "13:01",
		Items: []Item{
			{ShortDescription: "Mountain Dew 12PK", Price: "6.49"},
		},
		Total: "6.49",
	}

	body, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("Failed to marshal receipt: %v", err)
	}

	const numRequests = 10
	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/receipts/process", bytes.NewReader(body))
			w := httptest.NewRecorder()
			processReceiptsHandler(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status OK; got %v", resp.Status)
			}
		}()
	}

	wg.Wait()
}

// Verifies that points are calculated correctly for a receipt that triggers all rules
func TestCalculatePointsWithAllRules(t *testing.T) {
	receipt := Receipt{
		Retailer:     "BestBuy",
		PurchaseDate: "2022-01-01",
		PurchaseTime: "14:33",
		Items: []Item{
			{ShortDescription: "Mountain Dew 12PK", Price: "6.49"},
			{ShortDescription: "Emils Cheese Pizza", Price: "12.25"},
		},
		Total: "20.00",
	}

	expectedPoints := 7 + 50 + 25 + 5 + 2 + 6 + 10 // 7 (Best Buy) + 50 (round dollar) + 25 (multiple of 0.25) +
	// 5 (2 items on the list) + 2 (trimmed lengths) + 6 (odd day) + 10 (purchase time)
	points := calculatePoints(receipt)

	if points != expectedPoints {
		t.Errorf("Expected %d points; got %d", expectedPoints, points)
	}
}

// Ensure the expected points are correctly calculated for a receipt with large numbers
func TestCalculatePointsLargeNumbers(t *testing.T) {
	receipt := Receipt{
		Retailer: "Target",
		Items: []Item{
			{ShortDescription: "Expensive Item", Price: "999999.99"},
		},
		Total: "999999.99", // Ensure this is a round dollar amount
	}

	expectedPoints := 6 // 6 (Target)
	points := calculatePoints(receipt)

	if points < expectedPoints {
		t.Errorf("Expected at least %d points; got %d", expectedPoints, points)
	}
}

// Test a receipt with an empty list of items
func TestEmptyItemsList(t *testing.T) {
	receipt := Receipt{
		Retailer:     "Walmart",
		PurchaseDate: "2022-01-01",
		PurchaseTime: "13:01",
		Items:        []Item{},
		Total:        "10.00",
	}

	expectedPoints := 7 + 50 + 25 + 6 // 7 for retailer, 50 for round total, 25 for multiple of 0.25, 6 for odd day
	points := calculatePoints(receipt)

	if points != expectedPoints {
		t.Errorf("Expected %d points; got %d", expectedPoints, points)
	}
}

// Test a receipt where the retailer name contains no alphanumeric characters
func TestNonAlphanumericRetailerName(t *testing.T) {
	receipt := Receipt{
		Retailer:     "!@#$%^&*()",
		PurchaseDate: "2022-01-01",
		PurchaseTime: "13:01",
		Items: []Item{
			{ShortDescription: "Bread", Price: "2.50"},
		},
		Total: "2.50",
	}

	expectedPoints := 25 + 6 // 25 (multiple of 0.25) + 6 (odd day)
	points := calculatePoints(receipt)

	if points != expectedPoints {
		t.Errorf("Expected %d points; got %d", expectedPoints, points)
	}
}

// Test a receipt with multiple items have description lengths as multiples of 3
func TestMultipleItemsWithDescriptionLengthsMultipleOf3(t *testing.T) {
	receipt := Receipt{
		Retailer:     "GroceryStore",
		PurchaseDate: "2022-01-01",
		PurchaseTime: "13:01",
		Items: []Item{
			{ShortDescription: "ABC", Price: "1.00"},
			{ShortDescription: "DEF", Price: "2.00"},
		},
		Total: "3.00",
	}

	expectedPoints := 12 + 50 + 25 + 5 + 2 + 6 // 12 (GroceryStore) + 50 (round dollar) + 25 (multiple of 0.25)
	// + 5 (2 items on the list) + 2 (trimmed lengths) + 6 (odd day)
	points := calculatePoints(receipt)

	if points != expectedPoints {
		t.Errorf("Expected %d points; got %d", expectedPoints, points)
	}
}

// Test purchase times exactly at 2:00pm and 4:00pm
func TestBoundaryTimes(t *testing.T) {
	tests := []struct {
		time   string
		points int
	}{
		{"14:00", 96}, // Exactly 2:00 PM, should get points
		{"16:00", 86}, // Exactly 4:00 PM, should not get points
	}

	for _, tt := range tests {
		receipt := Receipt{
			Retailer:     "Store",
			PurchaseDate: "2022-01-01",
			PurchaseTime: tt.time,
			Items: []Item{
				{ShortDescription: "Item", Price: "1.00"},
			},
			Total: "1.00",
		}

		points := calculatePoints(receipt)

		if points != tt.points {
			t.Errorf("For time %s, expected %d points; got %d", tt.time, tt.points, points)
		}
	}
}

// Test a receipt with a total that is exactly a multiple of 0.25 but not a round dollar amount
func TestExactMultipleOfPoint25(t *testing.T) {
	receipt := Receipt{
		Retailer:     "Retail",
		PurchaseDate: "2022-01-01",
		PurchaseTime: "13:01",
		Items: []Item{
			{ShortDescription: "Item", Price: "0.50"},
		},
		Total: "0.50",
	}

	expectedPoints := 25 + 6 + 6 // 25 for multiple of 0.25, 6 for odd day, 6 for retailer name
	points := calculatePoints(receipt)

	if points != expectedPoints {
		t.Errorf("Expected %d points; got %d", expectedPoints, points)
	}
}

// Test items with special characters in their description
func TestSpecialCharactersInDescription(t *testing.T) {
	receipt := Receipt{
		Retailer:     "Shop",
		PurchaseDate: "2022-01-01",
		PurchaseTime: "13:01",
		Items: []Item{
			{ShortDescription: "!@# ABC", Price: "1.00"},
		},
		Total: "1.00",
	}

	expectedPoints := 4 + 50 + 25 + 6 + 1 // 4 (Shop) + 50 (round total) + 25 (multiple of 0.25) + 6 (odd day)
	// + 1 (item description rule)
	points := calculatePoints(receipt)

	if points != expectedPoints {
		t.Errorf("Expected %d points; got %d", expectedPoints, points)
	}
}

// Test with very high values to ensure no overflow or unexpected behavior
func TestMaximumValues(t *testing.T) {
	receipt := Receipt{
		Retailer:     "MaxStore",
		PurchaseDate: "2022-01-01",
		PurchaseTime: "14:33",
		Items: []Item{
			{ShortDescription: "Expensive Item", Price: "9999999999.99"},
		},
		Total: "9999999999.00",
	}

	expectedPoints := 8 + 50 + 25 + 6 + 10 // 8 (MaxStore) + 50 (round total) + 25 (multiple of 0.25)
	// + 6 (odd day) + 10 (time between 14:00-16:00)
	points := calculatePoints(receipt)

	if points != expectedPoints {
		t.Errorf("Expected %d points; got %d", expectedPoints, points)
	}
}

// Test receipts missing the purchase date or time
func TestMissingDateOrTime(t *testing.T) {
	tests := []struct {
		receipt Receipt
		valid   bool
	}{
		{Receipt{Retailer: "Store", PurchaseDate: "", PurchaseTime: "13:01",
			Items: []Item{{ShortDescription: "Item", Price: "1.00"}}, Total: "1.00"}, false},
		{Receipt{Retailer: "Store", PurchaseDate: "2022-01-01", PurchaseTime: "",
			Items: []Item{{ShortDescription: "Item", Price: "1.00"}}, Total: "1.00"}, false},
	}

	for _, tt := range tests {
		body, err := json.Marshal(tt.receipt)
		if err != nil {
			t.Fatalf("Failed to marshal receipt: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/receipts/process", bytes.NewReader(body))
		w := httptest.NewRecorder()
		processReceiptsHandler(w, req)

		resp := w.Result()
		if tt.valid && resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status OK; got %v", resp.Status)
		} else if !tt.valid && resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Expected status BadRequest; got %v", resp.Status)
		}
	}
}
