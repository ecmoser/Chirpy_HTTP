package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"time"

	auth "github.com/ecmoser/Chirpy_HTTP/internal/auth"
	"github.com/ecmoser/Chirpy_HTTP/internal/database"
	"github.com/google/uuid"
)

type chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Body string `json:"body"`
	}
	headers := r.Header
	token, err := auth.GetBearerToken(headers)
	if err != nil {
		respondWithError(w, 500, "Error getting bearer token")
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.tokenSecret)
	if err != nil {
		respondWithError(w, 401, "Error validating JWT: "+err.Error())
		return
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	rBody := request{}
	err = decoder.Decode(&rBody)
	if err != nil {
		respondWithError(w, 400, "Invalid request body")
		return
	}
	if len(rBody.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}
	clean_chirp := cleanChirp(rBody.Body)
	rawChirp, err := cfg.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   clean_chirp,
		UserID: userID,
	})
	if err != nil {
		respondWithError(w, 500, "Couldn't create chirp")
		return
	}
	c := chirp{
		ID:        rawChirp.ID,
		CreatedAt: rawChirp.CreatedAt,
		UpdatedAt: rawChirp.UpdatedAt,
		Body:      rawChirp.Body,
		UserID:    rawChirp.UserID,
	}
	respondWithJSON(w, 201, c)
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("author_id")
	if userID != "" {
		userID, err := uuid.Parse(userID)
		if err != nil {
			respondWithError(w, 400, "Invalid author_id")
			return
		}
		rawChirps, err := cfg.dbQueries.GetChirpsByUserID(r.Context(), userID)
		if err != nil {
			respondWithError(w, 500, "Couldn't get chirps")
			return
		}
		chirps := []chirp{}
		for _, rawChirp := range rawChirps {
			c := chirp{
				ID:        rawChirp.ID,
				CreatedAt: rawChirp.CreatedAt,
				UpdatedAt: rawChirp.UpdatedAt,
				Body:      rawChirp.Body,
				UserID:    rawChirp.UserID,
			}
			chirps = append(chirps, c)
		}
		respondWithJSON(w, 200, chirps)
		return
	}
	rawChirps, err := cfg.dbQueries.GetChirps(r.Context())
	if err != nil {
		respondWithError(w, 500, "Couldn't get chirps")
		return
	}
	chirps := []chirp{}
	for _, rawChirp := range rawChirps {
		c := chirp{
			ID:        rawChirp.ID,
			CreatedAt: rawChirp.CreatedAt,
			UpdatedAt: rawChirp.UpdatedAt,
			Body:      rawChirp.Body,
			UserID:    rawChirp.UserID,
		}
		chirps = append(chirps, c)
	}
	respondWithJSON(w, 200, chirps)
}

func (cfg *apiConfig) handlerGetChirpByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rawChirp, err := cfg.dbQueries.GetChirpByID(r.Context(), uuid.MustParse(id))
	if err != nil {
		respondWithError(w, 404, "Chirp not found")
		return
	}
	c := chirp{
		ID:        rawChirp.ID,
		CreatedAt: rawChirp.CreatedAt,
		UpdatedAt: rawChirp.UpdatedAt,
		Body:      rawChirp.Body,
		UserID:    rawChirp.UserID,
	}
	respondWithJSON(w, 200, c)
}

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	chirpID := r.PathValue("id")
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "No token found in header")
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.tokenSecret)
	if err != nil {
		respondWithError(w, 401, "Invalid token")
		return
	}
	chirp, err := cfg.dbQueries.GetChirpByID(r.Context(), uuid.MustParse(chirpID))
	if err != nil {
		respondWithError(w, 404, "Chirp not found")
		return
	}
	if chirp.UserID != userID {
		respondWithError(w, 403, "You are not allowed to delete this chirp")
		return
	}
	err = cfg.dbQueries.DeleteChirp(r.Context(), uuid.MustParse(chirpID))
	if err != nil {
		respondWithError(w, 500, "Couldn't delete chirp")
		return
	}
	w.WriteHeader(204)
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
