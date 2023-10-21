package main

import (
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
)

type Report struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	PhoneNumber string `gorm:"type:varchar(20)" json:"phone_number"`
	City        string `gorm:"type:varchar(50)" json:"city"`
	Where       string `gorm:"type:varchar(50)" json:"where"`
	HasPet      bool   `gorm:"type:boolean" json:"has_pet"`
	Additional  string `gorm:"type:varchar(255)" json:"additional"`

	PetID uint `gorm:"type:integer" json:"pet_id"`
}

type ReportResponse struct {
	ID          uint   `json:"id"`
	PhoneNumber string `json:"phone_number"`
	City        string `json:"city"`
	Where       string `json:"where"`
	HasPet      bool   `json:"has_pet"`
	Additional  string `json:"additional"`
}

func (r *Report) ToResponse() ReportResponse {
	return ReportResponse{
		ID:          r.ID,
		PhoneNumber: r.PhoneNumber,
		City:        r.City,
		Where:       r.Where,
		HasPet:      r.HasPet,
		Additional:  r.Additional,
	}
}

func CreateReport(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	petSlug := params["slug"]
	var report Report
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&report); err != nil {
		response := HTTPResponse{
			Error:  err.Error(),
			Status: http.StatusBadRequest,
		}
		RespondJson(w, r, response)
		return
	}

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
		RespondJson(w, r, response)
		return
	}

	log.Info().Msg(reqToken)
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
		RespondJson(w, r, response)
		return
	}

	claims, _ := readReportJWTClaims(token)

	var pet Pet
	db.Find(&pet, claims.id)

	if pet.Slug != petSlug {
		response := HTTPResponse{
			Error: FieldErrors{
				{
					Field: "jwt",
					Error: "Jeton d'authentification invalide",
				},
			},
			Status: http.StatusBadRequest,
		}
		RespondJson(w, r, response)
		return
	}

	log.Info().Msg(fmt.Sprintf("%v", claims.id))
	log.Info().Msg("JWT claims : " + fmt.Sprintf("%d", 10))
	if _, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		log.Info().Msg("JWT is valid")
	} else {
		response := HTTPResponse{
			Error: FieldErrors{
				{
					Field: "jwt",
					Error: "Jeton d'authentification invalide",
				},
			},
			Status: http.StatusBadRequest,
		}
		RespondJson(w, r, response)
		return
	}

	log.Info().
		Str("RequestID", r.Context().Value("requestID").(string)).
		Msg("Report received !")

	report.PetID = pet.ID
	if err := db.Create(&report).Error; err != nil {
		response := HTTPResponse{
			Error:  err.Error(),
			Status: http.StatusBadRequest,
		}
		RespondJson(w, r, response)
		return
	}

	response := HTTPResponse{
		Data:   report,
		Error:  nil,
		Status: http.StatusCreated,
	}
	RespondJson(w, r, response)
}
