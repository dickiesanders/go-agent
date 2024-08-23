package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"bytes"
	"strings"
)

const (
	validAPIToken = "9876543210"  // Replace with your actual token
	baseSQSEndpoint = "http://localhost:4100/queue/" // Base SQS endpoint
)

// Middleware to check API token
func checkAPIToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token != validAPIToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Handler to receive data and forward to the correct SQS queue
func receiveAndForwardToSQS(w http.ResponseWriter, r *http.Request) {
	// Extract the queue name from the URL path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Queue not specified", http.StatusBadRequest)
		return
	}
	queueName := parts[2]

	// Construct the SQS endpoint dynamically
	sqsEndpoint := baseSQSEndpoint + queueName

	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	// Forward the data to the GoAWS SQS queue
	resp, err := http.Post(sqsEndpoint, "application/x-www-form-urlencoded", bytes.NewReader(body))
	if err != nil {
		http.Error(w, "Failed to forward to SQS", http.StatusInternalServerError)
		log.Println("Error forwarding to SQS:", err)
		return
	}
	defer resp.Body.Close()

	// Respond to the client
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Data successfully forwarded to SQS queue: %s", queueName)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/receive/", receiveAndForwardToSQS) // The trailing slash allows dynamic paths

	// Apply token-checking middleware
	http.Handle("/receive/", checkAPIToken(mux)) // Apply to the entire path

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
