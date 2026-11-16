package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joeluismartinez/chirpy-go/internal/auth"
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

type chirpResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
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
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	req := email{}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		erroMessageHandler(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	hashedPwd, err := auth.HashPassword(req.Password)

	if err != nil {
		log.Printf("Error creating hash: %s", err)
		erroMessageHandler(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	insertNewUser, err := cfg.dbQueries.CreateUser(r.Context(), database.CreateUserParams{
		Email:          req.Email,
		HashedPassword: hashedPwd,
	})
	if err != nil {
		log.Printf("Error creating user: %s", err)
		erroMessageHandler(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response := userResponse{
		ID:        insertNewUser.ID,
		CreatedAt: insertNewUser.CreatedAt,
		UpdatedAt: insertNewUser.UpdatedAt,
		Email:     insertNewUser.Email,
	}

	respondJSON(w, http.StatusCreated, response)
}

func (cfg *apiConfig) loginUser(w http.ResponseWriter, r *http.Request) {
	type user struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	req := user{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		erroMessageHandler(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	storedPassword, err := cfg.dbQueries.GetUserPassword(r.Context(), req.Email)
	if err != nil {
		log.Printf("Error logging in: %s", err)
		erroMessageHandler(w, http.StatusInternalServerError, "Error retrieving password!")
	}

	match, err := auth.CheckPasswordHash(req.Password, storedPassword.HashedPassword)

	if err != nil {
		log.Printf("Error in matching passwords: %s", err)
		erroMessageHandler(w, http.StatusUnauthorized, "Passwords do not match!")
		return
	}

	response := userResponse{
		ID:        storedPassword.ID,
		CreatedAt: storedPassword.CreatedAt,
		UpdatedAt: storedPassword.UpdatedAt,
		Email:     storedPassword.Email,
	}

	if match {
		respondJSON(w, http.StatusOK, response)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
}

func (cfg *apiConfig) getChirp(w http.ResponseWriter, r *http.Request) {
	pathId := r.PathValue("id")
	id, err := uuid.Parse(pathId)
	if err != nil {
		erroMessageHandler(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	chirp, err := cfg.dbQueries.GetChirp(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			erroMessageHandler(w, http.StatusNotFound, "chirp not found!")
			return
		}

		erroMessageHandler(w, http.StatusBadRequest, "There was an error!")
		return
	}

	res := chirpResponse{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		UserID:    chirp.UserID,
		Body:      chirp.Body,
	}

	respondJSON(w, http.StatusOK, res)
}

func (cfg *apiConfig) getChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.dbQueries.GetChirps(r.Context())
	if err != nil {
		log.Printf("Error getting chirps: %s", err)
		erroMessageHandler(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	resp := []chirpResponse{}
	for _, chirp := range chirps {
		resp = append(resp, chirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		})
	}

	respondJSON(w, http.StatusOK, resp)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("\"OK\" (the text associated with the 200 status code)"))
}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	req := request{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		erroMessageHandler(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if req.UserID == uuid.Nil {
		erroMessageHandler(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if len(req.Body) > 140 {
		erroMessageHandler(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	chirp, err := cfg.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   profaneHandler(req.Body),
		UserID: req.UserID,
	})

	if err != nil {
		log.Printf("Error creating chirp: %s", err)
		erroMessageHandler(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	resp := chirpResponse{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}

	respondJSON(w, http.StatusCreated, resp)
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
		log.Fatalf("Error opening database: %s", err)
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
	mux.HandleFunc("POST /api/chirps", apiCfg.createChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.getChirps)
	mux.HandleFunc("GET /api/chirps/{id}", apiCfg.getChirp)
	mux.HandleFunc("POST /api/users", apiCfg.createUser)
	mux.HandleFunc("POST /api/login", apiCfg.loginUser)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	mux.Handle("/app/", http.StripPrefix("/app/", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	srv.ListenAndServe()
}
