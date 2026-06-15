package main

import (
	"log"
	"net/http"
)

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", handlerReadiness)

	mux.Handle(
		"/app/",
		http.StripPrefix(
			"/app/",
			http.FileServer(http.Dir(".")),
		),
	)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Fatal(server.ListenAndServe())
}
