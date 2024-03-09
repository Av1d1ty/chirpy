package main

import (
	"flag"
	"log"
	"net/http"
	"os"

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
	mux.HandleFunc("POST /api/login", apiCfg.postloginHandler)

	corsMux := middlewareCors(mux)
	log.Fatal(http.ListenAndServe(":8080", corsMux))
}

type apiConfig struct {
	fileserverHits int
	DB             *db.DB
}
