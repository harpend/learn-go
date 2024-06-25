package main

import (
	"bootdevserver/api"
	"log"
	"net/http"
)

func main() {
	var cfg api.ApiConfig
	db := api.DB{}
	log.Println("Connecting to Database.json...")
	db.ConnectDB("./database.json")
	mux := http.NewServeMux()
	mux.Handle("/app/*", cfg.MiddlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))

	mux.HandleFunc("GET /api/healthz", api.HandleHealthz)
	mux.HandleFunc("GET /admin/metrics", cfg.DisplayHits)
	mux.HandleFunc("/api/reset", cfg.ResetHits)
	mux.HandleFunc("POST /api/chirps", db.PostChirp)
	mux.HandleFunc("GET /api/chirps", db.GetChirps)
	mux.HandleFunc("GET /api/chirps/{id}", db.GetChirp)
	mux.HandleFunc("DELETE /api/chirps/{id}", db.DeleteChirp)
	mux.HandleFunc("POST /api/users", db.AddUser)
	mux.HandleFunc("PUT /api/users", db.EditUser)
	mux.HandleFunc("POST /api/login", db.Login)
	mux.HandleFunc("POST /api/refresh", db.Refresh)
	mux.HandleFunc("POST /api/revoke", db.Revoke)

	log.Println("Serving on port 8080...")
	http.ListenAndServe(":8080", mux)
}
