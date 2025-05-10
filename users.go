package main

import (
	"encoding/json"
	"net/http"
	"time"

	auth "github.com/ecmoser/Chirpy_HTTP/internal/auth"
	"github.com/ecmoser/Chirpy_HTTP/internal/database"
	"github.com/google/uuid"
)

type user struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	rBody := requestBody{}
	err := decoder.Decode(&rBody)
	if err != nil {
		respondWithError(w, 400, "Error decoding request body")
		return
	}
	hashed, err := auth.HashPassword(rBody.Password)
	if err != nil {
		respondWithError(w, 400, "Error hashing password")
		return
	}
	dbUser, err := cfg.dbQueries.CreateUser(r.Context(), database.CreateUserParams{
		Email:    rBody.Email,
		Password: hashed,
	})
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

func (cfg *apiConfig) handlerUserLogin(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		ExpiresIn int    `json:"expires_in_seconds"`
	}
	decoder := json.NewDecoder(r.Body)
	rBody := requestBody{}
	err := decoder.Decode(&rBody)
	if err != nil {
		respondWithError(w, 400, "Error decoding request body")
		return
	}
	if rBody.ExpiresIn == 0 || rBody.ExpiresIn > 3600 {
		rBody.ExpiresIn = 3600
	}
	userPassword, err := cfg.dbQueries.GetUserPassword(r.Context(), rBody.Email)
	if err != nil || auth.CheckPasswordHash(userPassword, rBody.Password) != nil {
		respondWithError(w, 401, "Invalid email or password")
		return
	}
	dbUser, err := cfg.dbQueries.GetUserByEmail(r.Context(), rBody.Email)
	if err != nil {
		respondWithError(w, 401, "Invalid email or password")
		return
	}
	token, err := auth.MakeJWT(dbUser.ID, cfg.tokenSecret, time.Duration(rBody.ExpiresIn)*time.Second)
	if err != nil {
		respondWithError(w, 500, "Error creating JWT")
		return
	}
	u := user{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
		Token:     token,
	}
	respondWithJSON(w, 200, u)
}
