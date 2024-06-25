package api

import (
	"fmt"
	"net/http"
	"text/template"
)

type ApiConfig struct {
	fileserverHits int
}
type data struct {
	Body string
}

func (cfg *ApiConfig) DisplayHits(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(405)
	}
	w.Header().Set("Content-Type", "text/html")
	body := fmt.Sprintf("Chirpy has been visited %d times!", cfg.fileserverHits)
	d := data{Body: body}
	t, err := template.ParseFiles("./api/metrics.html")
	if err != nil {
		fmt.Println(err)
	}
	t.Execute(w, d)
}
func (cfg *ApiConfig) ResetHits(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	cfg.fileserverHits = 0
}
func HandleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(405)
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *ApiConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}
