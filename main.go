package main

import (
	"fmt"
	"net/http"
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
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	message := fmt.Sprintf("Hits: %d", cfg.fileserverHits.Load())
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

func main() {
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	apiCfg := apiConfig{}

	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /reset", apiCfg.resetHandler)
	mux.HandleFunc("GET /metrics", apiCfg.metricsHandler)
	mux.Handle("/app/", http.StripPrefix("/app/", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	srv.ListenAndServe()
}
