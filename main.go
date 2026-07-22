package main

import (
	"net/http"
)

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

	mux.HandleFunc("GET /healthz", healthHandler)
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))
	srv.ListenAndServe()
}
