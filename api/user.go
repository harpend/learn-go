package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// dont store this in prod database use gitignore .env file instead same for firebase API key
var JWTSECRET string = "5vx6bK9ogyjjOjVXKUR0RcP0aOvS6nPE9uXbOn53EOd+9lyVZw7vvXU6pXy8qSHcwe0nKriYClrjCxnmBUP9Rg=="

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func findUser(email string, ch chan<- User, dbs DBStructure) {
	// use a for loop and a channel so that it can loop through and find the correct id for email
	for _, user := range dbs.Users {
		if user.Email == email {
			ch <- user
			return
		}
	}
	ch <- User{}
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	parts := strings.Split(authHeader, " ")
	return parts[1]
}

func (db *DB) EditUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		ID    int    `json:"id"`
		Email string `json:"email"`
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
	id, _ := strconv.Atoi(claims.Subject)
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		log.Println(err)
		w.WriteHeader(401)
		return
	}
	dbs, _ := db.loadDB()
	pswd, _ := bcrypt.GenerateFromPassword([]byte(params.Password), 10)
	dbs.Users[id] = User{
		ID:       id,
		Email:    params.Email,
		Password: string(pswd),
	}
	db.writeDB(dbs)
	resp := response{
		ID:    id,
		Email: params.Email,
	}
	dat, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}

func (db *DB) Refresh(w http.ResponseWriter, r *http.Request) {

	type response struct {
		Token string `json:"token"`
	}
	t := extractToken(r)
	dbs, _ := db.loadDB()
	exp, ok := dbs.Tokens[t]
	body, err := ioutil.ReadAll(r.Body)
	if !ok || !(exp.Expire.Time.After(time.Now())) || len(body) != 0 {
		log.Println(r.Body)
		w.WriteHeader(401)
		return
	}

	rc := jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(time.Hour))),
		Subject:   fmt.Sprint(exp.ID),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, rc)
	signedToken, err := token.SignedString([]byte(JWTSECRET))
	if err != nil {
		fmt.Println("Error signing token:", err)
		return
	}
	resp := response{
		Token: signedToken,
	}
	dat, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}

func (db *DB) Revoke(w http.ResponseWriter, r *http.Request) {
	t := extractToken(r)
	dbs, _ := db.loadDB()
	delete(dbs.Tokens, t)
	w.WriteHeader(204)
}

func (db *DB) Login(w http.ResponseWriter, r *http.Request) {
	// an optional json field looks like
	// 	Address *string `json:"address,omitempty"`

	type loginParams struct {
		Email    string        `json:"email"`
		Password string        `json:"password"`
		Expire   time.Duration `json:"expires_in_seconds,omitempty"`
	}
	type response struct {
		ID      int    `json:"id"`
		Email   string `json:"email"`
		Token   string `json:"token"`
		Refresh string `json:"refresh_token"`
	}
	// fill in registerclaims
	dbs, _ := db.loadDB()
	decoder := json.NewDecoder(r.Body)
	params := loginParams{}
	decoder.Decode(&params)
	log.Println(params)
	userChan := make(chan User)
	go findUser(params.Email, userChan, dbs)
	user := <-userChan
	if user == (User{}) {
		w.WriteHeader(401)
		return
	} else if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(params.Password)) != nil {
		w.WriteHeader(401)
		return
	}
	if params.Expire == 0 {
		params.Expire = time.Hour
	}
	rc := jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(params.Expire))),
		Subject:   fmt.Sprint(user.ID),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, rc)
	signedToken, err := token.SignedString([]byte(JWTSECRET))
	if err != nil {
		fmt.Println("Error signing token:", err)
		return
	}
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)
	refreshToken := hex.EncodeToString(randomBytes)
	resp := response{
		ID:      user.ID,
		Email:   user.Email,
		Token:   signedToken,
		Refresh: refreshToken,
	}
	duration := time.Hour * time.Duration(1440)
	newToken := ResfreshToken{
		Expire: *jwt.NewNumericDate(time.Now().Add(duration)),
		ID:     user.ID,
	}
	if dbs.Tokens == nil {
		dbs.Tokens = make(map[string]ResfreshToken)
	}
	dbs.Tokens[refreshToken] = newToken
	db.writeDB(dbs)
	dat, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)

}
func (db *DB) AddUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		User     string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		ID   int    `json:"id"`
		User string `json:"email"`
	}
	type errorParams struct {
		Error string `json:"error"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	dbs, _ := db.loadDB()
	err := decoder.Decode(&params)
	if err != nil {
		resp := errorParams{Error: "Something went wrong"}
		dat, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write(dat)
	}
	if len(params.User) <= 140 {
		if dbs.Users == nil {
			dbs.Users = make(map[int]User)
		}
		id := len(dbs.Users) + 1
		pswd, _ := bcrypt.GenerateFromPassword([]byte(params.Password), 10)
		sto := User{
			ID:       id,
			Email:    params.User,
			Password: string(pswd),
		}
		resp := response{
			ID:   id,
			User: params.User,
		}
		dbs.Users[id] = sto
		db.writeDB(dbs)
		dat, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write(dat)
	} else {
		resp := errorParams{Error: "Email is too long"}
		dat, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		w.Write(dat)
	}
}
