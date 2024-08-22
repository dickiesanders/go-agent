package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"bytes"
)

const (
	validAPIToken = "1234567890"  // Replace with your actual token
	sqsEndpoint   = "http://localhost:4100/queue/agent"
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

// Handler to receive data and forward to SQS
func receiveAndForwardToSQS(w http.ResponseWriter, r *http.Request) {
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

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Data successfully forwarded to SQS")
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/receive", receiveAndForwardToSQS)

	// Apply token-checking middleware
	http.Handle("/receive", checkAPIToken(mux))

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
