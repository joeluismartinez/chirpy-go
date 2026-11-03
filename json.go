package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func respondJSON(w http.ResponseWriter, status int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
}

func erroMessageHandler(w http.ResponseWriter, status int, message string) {
	type errorResponse struct {
		ErrorMessage string `json:"error"`
	}

	respondJSON(w, status, errorResponse{ErrorMessage: message})
}
