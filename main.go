package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func main() {
	mux := http.NewServeMux()
	apiCfg := &apiConfig{}
	fsHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fsHandler))
	mux.HandleFunc("GET /api/healthz", healthzHandler)
	mux.HandleFunc("GET /api/reset", apiCfg.resetHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("POST /api/validate_chirp", validateChirpHandler)
	corsMux := middlewareCors(mux)
	log.Fatal(http.ListenAndServe(":8080", corsMux))
}

type apiConfig struct {
	fileserverHits int
}

func middlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	page := `
    <html>
    <body>
        <h1>Welcome, Chirpy Admin</h1>
        <p>Chirpy has been visited %d times!</p>
    </body>
    </html>
    `
	w.Write([]byte(fmt.Sprintf(page, cfg.fileserverHits)))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits = 0
}

// Not used
func middlewareLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}
	type response struct {
		CleanedBody string `json:"cleaned_body"`
		// Error string `json:"error"`
	}
	chirpVal := chirp{}
	err := json.NewDecoder(r.Body).Decode(&chirpVal)
	switch {
	case err != nil:
        respondWithError(w, http.StatusInternalServerError,
                            fmt.Sprintf("Error decoding parameters: %s", err))
	case len(chirpVal.Body) > 140:
        respondWithError(w, http.StatusBadRequest, "Chirp is too long")
	default:
        resp := response{CleanedBody: censorProfanity(chirpVal.Body)}
        respJSON, _ := json.Marshal(resp)
		w.WriteHeader(200)
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(respJSON))
	}
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

func respondWithError(w http.ResponseWriter, code int, msg string) {
	if code > 499 {
		log.Printf("Responding with 5XX error: %s", msg)
	}
	type errorResponse struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(dat)
}
