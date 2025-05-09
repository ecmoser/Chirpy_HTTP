package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
)

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
