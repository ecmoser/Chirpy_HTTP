package main

import (
	"encoding/json"
	"net/http"

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
