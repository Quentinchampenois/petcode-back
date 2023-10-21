package main

import (
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Token struct {
	Email       string `json:"email"`
	TokenString string `json:"token"`
}

type ReportToken struct {
	PetID       uint   `json:"pet_id"`
	TokenString string `json:"token"`
}

func generateJWT(user *User) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)

	claims["authorized"] = true
	claims["user_id"] = user.ID
	claims["email"] = user.Email
	claims["exp"] = time.Now().Add(time.Minute * 60).Unix()
	tokenString, err := token.SignedString(getJWTSecret())

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func generateReportJWT(pet *Pet) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)

	claims["authorized"] = true
	claims["pet_id"] = pet.ID
	claims["exp"] = time.Now().Add(time.Minute * 60).Unix()
	tokenString, err := token.SignedString(getJWTSecret())

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func extractTokenFromJWT(token string) (*jwt.Token, error) {
	parse, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			log.Error().Msg("there was an error in parsing")
			return nil, fmt.Errorf("there was an error in parsing")
		}
		return getJWTSecret(), nil
	})

	if err != nil {
		return nil, err
	}

	return parse, nil
}

func readJWTClaims(token *jwt.Token) (*struct {
	id float64
}, error) {
	var userToken struct {
		id float64
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if claims["user_id"] == nil || claims["user_id"] == "" {
			return nil, fmt.Errorf("Missing required claims")
		}
		if claims["email"] == nil || claims["email"] == "" {
			return nil, fmt.Errorf("Missing required claims")
		}

		userToken.id = claims["user_id"].(float64)
	}

	return &userToken, nil
}

func readReportJWTClaims(token *jwt.Token) (*struct {
	id float64
}, error) {
	var reportToken struct {
		id float64
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if claims["pet_id"] == nil || claims["pet_id"] == "" {
			return nil, fmt.Errorf("Missing required claims")
		}

		reportToken.id = claims["pet_id"].(float64)
	}

	return &reportToken, nil
}

func isAuthorized(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqToken := r.Header.Get("Authorization")
		splitToken := strings.Split(reqToken, "Bearer ")
		reqToken = splitToken[1]
		if reqToken == "" {
			response := HTTPResponse{
				Error: FieldErrors{
					{
						Field: "jwt",
						Error: "Jeton d'authentification manquant",
					},
				},
				Status: http.StatusUnauthorized,
			}
			jsonResponse, _ := json.Marshal(response)
			w.WriteHeader(response.Status)
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write(jsonResponse)
			if err != nil {
				log.Error().Msg(err.Error())
			}
			return
		}

		token, err := extractTokenFromJWT(reqToken)
		if err != nil {
			response := HTTPResponse{
				Error: FieldErrors{
					{
						Field: "jwt",
						Error: err.Error(),
					},
				},
				Status: http.StatusBadRequest,
			}
			jsonResponse, _ := json.Marshal(response)
			w.WriteHeader(response.Status)
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write(jsonResponse)
			if err != nil {
				log.Error().Msg(err.Error())
			}
			return
		}

		if _, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			handler.ServeHTTP(w, r)
			return
		}

		response := HTTPResponse{
			Error: FieldErrors{
				{
					Field: "jwt",
					Error: "Jeton d'authentification invalide",
				},
			},
			Status: http.StatusBadRequest,
		}
		jsonResponse, _ := json.Marshal(response)
		w.WriteHeader(response.Status)
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(jsonResponse)
		if err != nil {
			log.Error().Msg(err.Error())
		}
	})
}

func getJWTSecret() []byte {
	if jwtSecret := os.Getenv("JWT_SECRET_KEY"); jwtSecret != "" {
		return []byte(jwtSecret)
	}

	log.Fatal().Msg("You must define 'JWT_SECRET_KEY' environment variable for JWT authentication system")
	return nil
}
