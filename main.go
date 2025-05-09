package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ecmoser/Chirpy_HTTP/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
}

type user struct {
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

func respondWithError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(fmt.Appendf(nil, `{"error": "%s"}`, msg))
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		respondWithError(w, 500, "Internal server error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func cleanChirp(chirp string) string {
	dirty_words := []string{"kerfuffle", "sharbert", "fornax"}
	clean_chirp := ""
	for word := range strings.SplitSeq(chirp, " ") {
		if slices.Contains(dirty_words, strings.ToLower(word)) {
			clean_chirp += "**** "
		} else {
			clean_chirp += word + " "
		}
	}
	return strings.TrimSpace(clean_chirp)
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(fmt.Appendf(nil, `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load()))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, 403, "Forbidden")
		return
	}
	cfg.fileserverHits.Store(0)
	err := cfg.dbQueries.ClearUsers(r.Context())
	if err != nil {
		respondWithError(w, 500, "Error clearing users")
		return
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("OK"))
}

func handlerHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handlerUsers(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	rBody := requestBody{}
	err := decoder.Decode(&rBody)
	if err != nil {
		respondWithError(w, 400, "Error decoding request body")
		return
	}
	dbUser, err := cfg.dbQueries.CreateUser(r.Context(), rBody.Email)
	u := user{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}
	if err != nil {
		respondWithError(w, 500, "Error creating user")
		return
	}
	respondWithJSON(w, 201, u)
}

func (cfg *apiConfig) handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	rBody := requestBody{}
	err := decoder.Decode(&rBody)
	if err != nil {
		respondWithError(w, 500, "Error decoding request body")
		return
	}
	if len(rBody.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}
	clean_chirp := cleanChirp(rBody.Body)
	respondWithJSON(w, 200, map[string]any{"valid": true, "cleaned_body": clean_chirp})
}

func main() {
	godotenv.Load()

	db_url := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", db_url)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)

	const filepathRoot = "./app/"
	const port = "8080"
	apiCfg := apiConfig{
		dbQueries: dbQueries,
		platform:  os.Getenv("PLATFORM"),
	}

	mux := http.NewServeMux()

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))))
	mux.HandleFunc("GET /api/healthz", handlerHealthz)
	mux.HandleFunc("POST /api/validate_chirp", apiCfg.handlerValidateChirp)
	mux.HandleFunc("POST /api/users", apiCfg.handlerUsers)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %v\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())

}
