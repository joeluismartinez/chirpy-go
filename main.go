package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	message := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits.Load())
	w.Write([]byte(message))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	message := fmt.Sprintf("Hits: %d", cfg.fileserverHits.Load())
	cfg.fileserverHits.Store(0)
	w.Write([]byte(message))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("\"OK\" (the text associated with the 200 status code)"))
}

func erroMessageHandler(w http.ResponseWriter, status int, message string) {

	type errorResponse struct {
		ErrorMessage string `json:"error"`
	}

	msg := errorResponse{
		ErrorMessage: message,
	}

	m, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-TYpe", "application/json")
	w.WriteHeader(status)
	w.Write(m)
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	type resp struct {
		Valid string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		erroMessageHandler(w, 500, "something went wrong")
		return
	}

	if len(params.Body) > 140 {
		erroMessageHandler(w, 400, "Chirp is too long")
		return
	}

	cleanedBody := profaneHandler(params.Body)

	response := resp{
		Valid: cleanedBody,
	}

	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func profaneHandler(message string) string {

	type resp struct {
		Data string
	}

	profaneListy := []string{"kerfuffle", "sharbert", "fornax"}
	iterMessage := strings.Split(message, " ")
	for i, msg := range iterMessage {
		for _, profanity := range profaneListy {
			lowerMsg := strings.ToLower(msg)
			lowerPrfn := strings.ToLower(profanity)
			if lowerMsg == lowerPrfn {
				iterMessage[i] = "****"
			}
		}
	}
	censored := strings.Join(iterMessage, " ")
	return censored
}

func main() {
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	apiCfg := apiConfig{}

	mux.HandleFunc("GET /api/healthz", healthHandler)
	mux.HandleFunc("POST /api/validate_chirp", validateHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	mux.Handle("/app/", http.StripPrefix("/app/", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	srv.ListenAndServe()
}
