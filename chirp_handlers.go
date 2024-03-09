package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Av1d1ty/chirpy/internal/db"
)

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.DB.GetChirps()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError,
			fmt.Sprintf("Error getting chirps: %s", err))
		return
	}
	respondWithJSON(w, 200, chirps)
}

func (cfg *apiConfig) getChirpHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest,
			fmt.Sprintf("Error parsing ID: %s", err))
		return
	}
	chirp, err := cfg.DB.GetChirp(id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError,
			fmt.Sprintf("Error getting chirp: %s", err))
		return
	}
	if chirp == (db.Chirp{}) {
		respondWithError(w, http.StatusNotFound, "Chirp not found")
		return
	}
	respondWithJSON(w, 200, chirp)
}

func (cfg *apiConfig) postChirpHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	params := parameters{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError,
			fmt.Sprintf("Error decoding parameters: %s", err))
	}
	if len(params.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}
	cleanedBody := censorProfanity(params.Body)
	chirp, err := cfg.DB.CreateChirp(cleanedBody)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError,
			fmt.Sprintf("Error creating chirp: %s", err))
		return
	}
	respondWithJSON(w, 201, chirp)
}

func censorProfanity(body string) string {
	words := strings.Split(body, " ")
	for i, word := range words {
		if isProfanity(strings.ToLower(word)) {
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}

func isProfanity(word string) bool {
	profaneWords := []string{"kerfuffle", "sharbert", "fornax"}
	for _, profane := range profaneWords {
		if word == profane {
			return true
		}
	}
	return false
}
