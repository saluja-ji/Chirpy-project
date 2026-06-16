package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

type Chirp struct {
	Body string `json:"body"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type CleanedResponse struct {
	CleanedBody string `json:"cleaned_body"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	html := fmt.Sprintf(`
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())

	w.Write([]byte(html))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	var chirp Chirp

	err := json.NewDecoder(r.Body).Decode(&chirp)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "Something went wrong",
		})
		return
	}

	if len(chirp.Body) > 140 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "Chirp is too long",
		})
		return
	}

	words := strings.Split(chirp.Body, " ")

	for i, word := range words {
		switch strings.ToLower(word) {
		case "kerfuffle", "sharbert", "fornax":
			words[i] = "****"
		}
	}

	cleaned := strings.Join(words, " ")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(CleanedResponse{
		CleanedBody: cleaned,
	})
}

func main() {
	apiCfg := &apiConfig{}

	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/healthz", handlerReadiness)
	mux.HandleFunc("/api/validate_chirp", handlerValidateChirp)

	// Admin endpoints
	mux.HandleFunc("/admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("/admin/reset", apiCfg.handlerReset)

	// File server
	fileServer := http.StripPrefix(
		"/app/",
		http.FileServer(http.Dir(".")),
	)

	mux.Handle(
		"/app/",
		apiCfg.middlewareMetricsInc(fileServer),
	)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Starting server on :8080")

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
