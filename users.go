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
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
	AccessToken  string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	defer r.Body.Close()
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
		ID:          dbUser.ID,
		CreatedAt:   dbUser.CreatedAt,
		UpdatedAt:   dbUser.UpdatedAt,
		Email:       dbUser.Email,
		IsChirpyRed: dbUser.IsChirpyRed,
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
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	rBody := requestBody{}
	err := decoder.Decode(&rBody)
	if err != nil {
		respondWithError(w, 400, "Error decoding request body")
		return
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
	token, err := auth.MakeJWT(dbUser.ID, cfg.tokenSecret, time.Duration(1)*time.Hour)
	if err != nil {
		respondWithError(w, 500, "Error creating JWT")
		return
	}
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, 500, "Error creating refresh token")
		return
	}
	_, err = cfg.dbQueries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:  refreshToken,
		UserID: dbUser.ID,
	})
	if err != nil {
		respondWithError(w, 500, "Error saving refresh token")
		return
	}
	u := user{
		ID:           dbUser.ID,
		CreatedAt:    dbUser.CreatedAt,
		UpdatedAt:    dbUser.UpdatedAt,
		Email:        dbUser.Email,
		IsChirpyRed:  dbUser.IsChirpyRed,
		AccessToken:  token,
		RefreshToken: refreshToken,
	}
	respondWithJSON(w, 200, u)
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Token string `json:"token"`
	}{
		Token: "",
	}
	rHeader := r.Header
	token, err := auth.GetBearerToken(rHeader)
	if err != nil {
		respondWithError(w, 401, "No token found in header")
		return
	}
	userID, err := cfg.dbQueries.GetUserFromRefreshToken(r.Context(), token)
	if err != nil {
		respondWithError(w, 401, "Invalid refresh token")
		return
	}
	accessToken, err := auth.MakeJWT(userID, cfg.tokenSecret, time.Duration(1)*time.Hour)
	if err != nil {
		respondWithError(w, 500, "Error creating access token")
		return
	}
	response.Token = accessToken
	respondWithJSON(w, 200, response)
}

func (cfg *apiConfig) handlerRevokeRefreshToken(w http.ResponseWriter, r *http.Request) {
	rHeader := r.Header
	token, err := auth.GetBearerToken(rHeader)
	if err != nil {
		respondWithError(w, 401, "No token found in header")
		return
	}
	err = cfg.dbQueries.RevokeRefreshToken(r.Context(), token)
	if err != nil {
		respondWithError(w, 500, "Error revoking refresh token")
		return
	}
	w.WriteHeader(204)
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	header := r.Header
	token, err := auth.GetBearerToken(header)
	if err != nil {
		respondWithError(w, 401, "No token found in header")
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.tokenSecret)
	if err != nil {
		respondWithError(w, 401, "Invalid token")
		return
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	rBody := requestBody{}
	err = decoder.Decode(&rBody)
	if err != nil {
		respondWithError(w, 400, "Error decoding request body")
		return
	}
	hashed, err := auth.HashPassword(rBody.Password)
	if err != nil {
		respondWithError(w, 400, "Error hashing password")
		return
	}
	dbUser, err := cfg.dbQueries.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:       userID,
		Email:    rBody.Email,
		Password: hashed,
	})
	if err != nil {
		respondWithError(w, 500, "Error updating user")
		return
	}
	respondWithJSON(w, 200, struct {
		ID          uuid.UUID `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Email       string    `json:"email"`
		IsChirpyRed bool      `json:"is_chirpy_red"`
	}{
		ID:          dbUser.ID,
		CreatedAt:   dbUser.CreatedAt,
		UpdatedAt:   dbUser.UpdatedAt,
		Email:       dbUser.Email,
		IsChirpyRed: dbUser.IsChirpyRed,
	})
}
