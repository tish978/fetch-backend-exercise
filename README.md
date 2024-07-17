# fetch-backend-exercise
This repo represents my submission for the Backend Take Home Exercise from Fetch.

## Overview 

This application processes receipts and calculates points based on specific rules. It provides a REST API to process receipts and retreive points.

## Endpoints

### Process Receipts
- **Path**: `/receipts/process`
- **Method**: `POST`
- **Payload**: JSON object representing the receipt.
- **Response**: JSON object containing the receipt ID.

### Get Points
- **Path**: `/receipts/{id}/points`
- **Method**: `GET`
- **Response**: JSON object containing the points awarded for the receipt.

## Running the Application
1. Clone the repository
2. In the project directory run "go run main.go"
3. Execute unit tests by running "go test ./..." in the project directory
