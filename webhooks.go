package main

import (
	"encoding/json"
	"net/http"

	auth "github.com/ecmoser/Chirpy_HTTP/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerPolkaWebhook(w http.ResponseWriter, r *http.Request) {
	requestBody := struct {
		Event string `json:"event"`
		Data  struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}{}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&requestBody)
	if err != nil {
		respondWithError(w, 400, "Error decoding request body")
		return
	}
	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}
	if apiKey != cfg.polkaApiKey {
		respondWithError(w, 401, "Invalid API key")
		return
	}
	if requestBody.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}
	err = cfg.dbQueries.UpdateToChirpyRed(r.Context(), requestBody.Data.UserID)
	if err != nil {
		respondWithError(w, 404, "Error updating to Chirpy Red")
		return
	}
	w.WriteHeader(204)
}
