package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Av1d1ty/chirpy/internal/db"
)

func main() {
    dbg := flag.Bool("debug", false, "Enable debug mode")
    flag.Parse()
    if *dbg {
        log.Println("Debug mode enabled")
        os.Remove("database.json")
    }
    dbFile, err := db.NewDB("database.json")
    if err != nil {
        log.Fatalf("Error opening database: %s", err)
        return
    }

	mux := http.NewServeMux()
	apiCfg := &apiConfig{DB: dbFile}
	fsHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fsHandler))
	mux.HandleFunc("GET /api/healthz", healthzHandler)
	mux.HandleFunc("GET /api/reset", apiCfg.resetHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("GET /api/chirps", apiCfg.getChirpsHandler)
	mux.HandleFunc("GET /api/chirps/{id}", apiCfg.getChirpHandler)
	mux.HandleFunc("POST /api/chirps", apiCfg.postChirpHandler)
	mux.HandleFunc("POST /api/users", apiCfg.postUserHandler)
	corsMux := middlewareCors(mux)
	log.Fatal(http.ListenAndServe(":8080", corsMux))
}

type apiConfig struct {
	fileserverHits int
    DB *db.DB
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

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, r *http.Request) {
    chirps, err := cfg.DB.GetChirps()
    if err != nil {
        respondWithError(w, http.StatusInternalServerError,
            fmt.Sprintf("Error getting chirps: %s", err))
        return
    }
    respJSON, _ := json.Marshal(chirps)
    w.WriteHeader(200)
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(respJSON))
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
    respJSON, _ := json.Marshal(chirp)
    w.WriteHeader(200)
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(respJSON))
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
    respJSON, _ := json.Marshal(chirp)
    w.WriteHeader(201)
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(respJSON))
}

func (cfg *apiConfig) postUserHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}
	params := parameters{}
	err := json.NewDecoder(r.Body).Decode(&params)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError,
                         fmt.Sprintf("Error decoding parameters: %s", err))
    }
    user, err := cfg.DB.CreateUser(params.Email)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError,
                         fmt.Sprintf("Error creating chirp: %s", err))
        return
    }
    respJSON, _ := json.Marshal(user)
    w.WriteHeader(201)
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(respJSON))
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
