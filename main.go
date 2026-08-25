package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joeluismartinez/chirpy-go/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
}

type userResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
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
	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	cfg.dbQueries.DeleteUsers(r.Context())
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	message := fmt.Sprintf("Hits: %d", cfg.fileserverHits.Load())
	cfg.fileserverHits.Store(0)
	w.Write([]byte(message))
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type email struct {
		Email string `json:"email"`
	}

	req := email{}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	insertNewUser, err := cfg.dbQueries.CreateUser(r.Context(), req.Email)
	if err != nil {
		log.Printf("Error creating user.", err)
		return
	}

	response := userResponse{
		ID:        insertNewUser.ID,
		CreatedAt: insertNewUser.CreatedAt,
		UpdatedAt: insertNewUser.UpdatedAt,
		Email:     insertNewUser.Email,
	}

	data, err := json.Marshal(response)

	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)

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
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		fmt.Errorf("DB is fucked up")
		return
	}

	dbQueries := database.New(db)

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	apiCfg := apiConfig{
		dbQueries: dbQueries,
		platform:  os.Getenv("PLATFORM"),
	}

	mux.HandleFunc("GET /api/healthz", healthHandler)
	mux.HandleFunc("POST /api/validate_chirp", validateHandler)
	mux.HandleFunc("POST /api/users", apiCfg.createUser)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	mux.Handle("/app/", http.StripPrefix("/app/", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	srv.ListenAndServe()
}
