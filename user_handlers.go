package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

func (cfg *apiConfig) putUserHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
    type response struct {
        Id    int    `json:"id"`
        Email string `json:"email"`
    }
    jwtHeader := r.Header.Get("Authorization")
    if jwtHeader == "" {
        respondWithError(w, http.StatusUnauthorized, "No JWT present")
        return
    }
    jwtToken := jwtHeader[len("Bearer "):]
    token, err := jwt.ParseWithClaims(jwtToken, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(cfg.jwtSecret), nil
    })
    if err != nil || !token.Valid {
        respondWithError(w, http.StatusUnauthorized, "Invalid JWT")
        return
    }
    params := parameters{}
    if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
        respondWithError(w, http.StatusInternalServerError,
            fmt.Sprintf("Error decoding parameters: %s", err))
        return
    }
    userId := token.Claims.(*jwt.StandardClaims).Subject
    if userId == "" {
        respondWithError(w, http.StatusUnauthorized, "Invalid JWT")
        return
    }
    userIdInt, err := strconv.Atoi(userId)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError,
            fmt.Sprintf("Error converting user ID to int: %s", err))
        return
    }
    user, err := cfg.DB.GetUser(userIdInt)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError,
            fmt.Sprintf("Error getting user: %s", err))
        return
    }
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't hash password")
		return
	}
    user.Email = params.Email
    user.HashedPassword = string(hashedPassword)
    if err := cfg.DB.UpdateUser(user); err != nil {
        respondWithError(w, http.StatusInternalServerError,
            fmt.Sprintf("Error updating user: %s", err))
        return
    }
    respondWithJSON(w, 200, response{user.Id, user.Email})
}

func (cfg *apiConfig) postUserHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		Id    int    `json:"id"`
		Email string `json:"email"`
	}
	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusInternalServerError,
			fmt.Sprintf("Error decoding parameters: %s", err))
	}
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
    if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't hash password")
		return
	}

	user, err := cfg.DB.CreateUser(params.Email, string(hashedPassword))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError,
			fmt.Sprintf("Error creating user: %s", err))
		return
	}
	respondWithJSON(w, 201, response{user.Id, user.Email})
}

func (cfg *apiConfig) postLoginHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email            string `json:"email"`
		Password         string `json:"password"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
	}
	type response struct {
		Id    int    `json:"id"`
		Email string `json:"email"`
        Token string `json:"token"`
	}
	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusInternalServerError,
			fmt.Sprintf("Error decoding parameters: %s", err))
		return
	}
	user, err := cfg.DB.GetUserByEmail(params.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError,
			fmt.Sprintf("Error getting user: %s", err))
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(params.Password))
	if err != nil {
		respondWithError(w, http.StatusUnauthorized,
			fmt.Sprintf("Invalid password"))
		return
	}
    token, err := generateJWT(cfg.jwtSecret, user.Id, params.ExpiresInSeconds)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError,
            fmt.Sprintf("Error generating JWT: %s", err))
        return
    }
	respondWithJSON(w, 200, response{user.Id, user.Email, token})
}

func generateJWT(secret string, id int, expiresInSeconds int) (string, error) {
    if expiresInSeconds <= 0 {
        expiresInSeconds = int(time.Hour * 24 / time.Second)
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
        Issuer:    "chirpy",
        Subject:   strconv.Itoa(id),
        IssuedAt:  time.Now().Unix(),
        ExpiresAt: time.Now().Add(time.Duration(expiresInSeconds) * time.Second).Unix(),
    })
    return token.SignedString([]byte(secret))
}
