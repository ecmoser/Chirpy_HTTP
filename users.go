package main

import (
	"encoding/json"
	"net/http"
)

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
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
