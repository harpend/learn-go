package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt/v4"
)

type DB struct {
	path string
	mux  *sync.RWMutex
}

type Chirp struct {
	ID     int    `json:"id"`
	Author int    `json:"author_id"`
	Body   string `json:"body"`
}

type ResfreshToken struct {
	Expire jwt.NumericDate `json:"expiration_time"`
	ID     int             `json:"id"`
}

type DBStructure struct {
	Chirps map[int]Chirp            `json:"chirps"`
	Users  map[int]User             `json:"users"`
	Tokens map[string]ResfreshToken `json:"tokens"`
}

func (db *DB) ConnectDB(path string) (*DB, error) {
	_, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Creating database.json...")
			os.Create(path)
			dat, _ := json.Marshal(DBStructure{})
			os.WriteFile(path, dat, 0666)
		} else {
			log.Fatal(err)
		}
	}
	db.path = path
	db.mux = &sync.RWMutex{}
	return db, nil
}

func (db *DB) loadDB() (DBStructure, error) {
	db.mux.RLock()
	defer db.mux.RUnlock()
	dat, err := os.ReadFile(db.path)
	if err != nil {
		return DBStructure{}, err
	}
	dbs := DBStructure{}
	json.Unmarshal(dat, &dbs)
	return dbs, nil
}
func (db *DB) writeDB(dbStructure DBStructure) error {
	db.mux.Lock()
	defer db.mux.Unlock()
	dat, err := json.Marshal(dbStructure)
	if err != nil {
		return err
	}
	err = os.WriteFile(db.path, dat, 0666)
	return err
}
func (db *DB) PostChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Chirp string `json:"body"`
	}
	type errorParams struct {
		Error string `json:"error"`
	}
	t := extractToken(r)
	token, err := jwt.ParseWithClaims(t, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(JWTSECRET), nil
	})
	if err != nil {
		log.Println(err)
		w.WriteHeader(401)
		return
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		log.Println(err)
		w.WriteHeader(401)
		return
	}
	authID, _ := strconv.Atoi(claims.Subject)
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	dbs, _ := db.loadDB()
	err = decoder.Decode(&params)
	if err != nil {
		resp := errorParams{Error: "Something went wrong"}
		dat, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write(dat)
	}
	if len(params.Chirp) <= 140 {
		sa := strings.Split(params.Chirp, " ")
		for i, s := range sa {
			if strings.ToLower(s) == "kerfuffle" || strings.ToLower(s) == "sharbert" || strings.ToLower(s) == "fornax" {
				sa[i] = "****"
			} else {
				sa[i] = s
			}
		}
		if dbs.Chirps == nil {
			dbs.Chirps = make(map[int]Chirp)
		}

		s := strings.Join(sa, " ")
		id := len(dbs.Chirps) + 1
		resp := Chirp{
			ID:     id,
			Author: authID,
			Body:   s,
		}
		dbs.Chirps[id] = resp
		db.writeDB(dbs)
		dat, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write(dat)
	} else {
		resp := errorParams{Error: "Chirp is too long"}
		dat, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		w.Write(dat)
	}
}
func (db *DB) GetChirps(w http.ResponseWriter, r *http.Request) {
	dbs, err := db.loadDB()
	if err != nil {
		log.Println(err)
	}
	var chirps []Chirp
	for _, chirp := range dbs.Chirps {
		chirps = append(chirps, chirp)
	}

	dat, err := json.Marshal(chirps)
	if err != nil {
		log.Println(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}

func (db *DB) GetChirp(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	dbs, _ := db.loadDB()
	chirp, ok := dbs.Chirps[id]
	if !ok {
		w.WriteHeader(404)
		return
	}
	dat, err := json.Marshal(chirp)
	if err != nil {
		log.Println(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}

func (db *DB) DeleteChirp(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	dbs, _ := db.loadDB()
	chirp, ok := dbs.Chirps[id]
	if !ok {
		w.WriteHeader(404)
		return
	}
	t := extractToken(r)
	token, err := jwt.ParseWithClaims(t, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(JWTSECRET), nil
	})
	if err != nil {
		log.Println(err)
		w.WriteHeader(401)
		return
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		log.Println(err)
		w.WriteHeader(401)
		return
	}
	id2, _ := strconv.Atoi(claims.Subject)
	if chirp.Author != id2 {
		w.WriteHeader(403)
		return
	}
	delete(dbs.Chirps, id)
	w.WriteHeader(204)
}
