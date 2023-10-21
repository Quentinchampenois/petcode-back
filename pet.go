package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Pet struct {
	gorm.Model
	ID        uint   `gorm:"primaryKey" json:"ID"`
	Name      string `gorm:"type:varchar(30)" json:"name"`
	Breed     string `json:"breed"`
	Sexe      string `gorm:"type:varchar(10);CHECK(sexe IN ('male', 'female'))" json:"sexe"`
	Birthdate string `gorm:"type:varchar(12)" json:"birthdate"`
	Slug      string `gorm:"type:varchar(40);unique" json:"slug"`
	UserID    uint   `json:"user_id" gorm:"foreignKey:ID"`
	User      User   `json:"-"`
	QRCodeID  uint   `json:"qrcode_id"`
	QRCode    QRCode `json:"qrcode"`
}

func (p *Pet) BeforeCreate(tx *gorm.DB) (err error) {
	p.Slug = (uuid.New()).String()
	return
}

func (p *Pet) AfterSave(tx *gorm.DB) (err error) {
	if p.QRCodeID != 0 {
		return
	}

	url := fmt.Sprintf("%s/pet/%s", os.Getenv("FRONTEND_URL"), p.Slug)
	qrCode := QRCode{
		Url:    url,
		Base64: GenerateBase64(url),
		PetID:  p.ID,
	}

	tx.Create(&qrCode)
	p.QRCodeID = qrCode.ID
	tx.Save(p)
	return
}

func (p *Pet) Validate() FieldErrors {
	var fieldErr FieldErrors

	// Name is required
	if p.Name == "" {
		fieldErr = append(fieldErr, FieldError{
			Field: "name",
			Error: "Le nom est obligatoire",
		})
	}

	// Breed is required
	if p.Breed == "" {
		fieldErr = append(fieldErr, FieldError{
			Field: "breed",
			Error: "La race est obligatoire",
		})
	}

	// Sexe is required
	if p.Sexe != "male" && p.Sexe != "female" {
		fieldErr = append(fieldErr, FieldError{
			Field: "sexe",
			Error: "Est-ce un male ou une femelle ?",
		})
	}

	// Birthdate is required
	if p.Birthdate == "" {
		fieldErr = append(fieldErr, FieldError{
			Field: "birthdate",
			Error: "La date de naissance est obligatoire",
		})
	}

	return fieldErr
}

func UpdatePet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	petSlug := params["slug"]

	reqToken := r.Header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer ")
	reqToken = splitToken[1]
	// Find user id in JWT token
	token, err := extractTokenFromJWT(reqToken)
	if err != nil {
		response := HTTPResponse{
			Error: FieldErrors{
				FieldError{
					Field: "-",
					Error: err.Error(),
				},
			},
			Status: http.StatusBadRequest,
		}
		RespondJson(w, r, response)
		return
	}
	userToken, _ := readJWTClaims(token)

	if !isValidUUID(petSlug) {
		response := HTTPResponse{
			Error: FieldErrors{
				{
					Field: "slug",
					Error: "L'identifiant de votre animal semble invalide",
				},
			},
			Status: http.StatusBadRequest,
		}
		RespondJson(w, r, response)
		return
	}

	var incomingPet Pet
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&incomingPet); err != nil {
		response := HTTPResponse{
			Error: FieldErrors{
				FieldError{
					Field: "-",
					Error: err.Error(),
				},
			},
			Status: http.StatusBadRequest,
		}
		RespondJson(w, r, response)
		return
	}

	errors := incomingPet.Validate()
	if len(errors) > 0 {
		response := HTTPResponse{
			Data:   incomingPet,
			Error:  errors,
			Status: http.StatusUnprocessableEntity,
		}
		RespondJson(w, r, response)
		return
	}

	var existingPet Pet
	query := db.Where("user_id = ?", userToken.id).Where("slug = ?", petSlug).First(&existingPet)
	if query.Error != nil {
		response := HTTPResponse{
			Data: nil,
			Error: FieldErrors{
				FieldError{
					Field: "-",
					Error: query.Error.Error(),
				},
			},
			Status: http.StatusUnauthorized,
		}
		RespondJson(w, r, response)
		return
	}

	// Update fields based on the incoming payload
	existingPet.Name = incomingPet.Name
	existingPet.Breed = incomingPet.Breed
	existingPet.Birthdate = incomingPet.Birthdate
	existingPet.Sexe = incomingPet.Sexe

	// Begin a new transaction
	tx := db.Begin()

	// Update the existing Pet record
	if err := tx.Save(&existingPet).Error; err != nil {
		response := HTTPResponse{
			Data: nil,
			Error: FieldErrors{
				FieldError{
					Field: "-",
					Error: query.Error.Error(),
				},
			},
			Status: http.StatusUnprocessableEntity,
		}
		RespondJson(w, r, response)

		tx.Rollback()
		return
	}

	tx.Commit()

	response := HTTPResponse{
		Data:   existingPet,
		Error:  nil,
		Status: http.StatusOK,
	}

	RespondJson(w, r, response)
}

func CreatePet(w http.ResponseWriter, r *http.Request) {
	var pet Pet

	reqToken := r.Header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer ")
	reqToken = splitToken[1]
	// Find user id in JWT token
	token, err := extractTokenFromJWT(reqToken)
	if err != nil {
		response := HTTPResponse{
			Error: FieldErrors{
				FieldError{
					Field: "-",
					Error: err.Error(),
				},
			},
			Status: http.StatusBadRequest,
		}
		RespondJson(w, r, response)
		return
	}
	userToken, _ := readJWTClaims(token)

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&pet); err != nil {
		response := HTTPResponse{
			Error:  err.Error(),
			Status: http.StatusBadRequest,
		}
		RespondJson(w, r, response)
		return
	}

	pet.UserID = uint(userToken.id)
	errors := pet.Validate()

	if len(errors) > 0 {
		response := HTTPResponse{
			Data:   pet,
			Error:  errors,
			Status: http.StatusUnprocessableEntity,
		}

		RespondJson(w, r, response)
		return
	}
	// Create a new pet record
	db.Create(&pet)

	response := HTTPResponse{
		Data:   pet,
		Error:  nil,
		Status: http.StatusCreated,
	}

	RespondJson(w, r, response)
}

func GetPets(w http.ResponseWriter, r *http.Request) {
	var pets []Pet
	reqToken := r.Header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer ")
	reqToken = splitToken[1]
	// Find user id in JWT token
	token, err := extractTokenFromJWT(reqToken)
	if err != nil {
		response := HTTPResponse{
			Error: FieldErrors{
				FieldError{
					Field: "-",
					Error: err.Error(),
				},
			},
			Status: http.StatusBadRequest,
		}
		RespondJson(w, r, response)
		return
	}
	userToken, _ := readJWTClaims(token)

	db.Where("user_id = ?", userToken.id).Find(&pets)

	response := HTTPResponse{
		Data:   pets,
		Error:  nil,
		Status: http.StatusOK,
	}

	RespondJson(w, r, response)
}

func GetPetBySlug(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	petSlug := params["slug"]

	reqToken := r.Header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer ")
	reqToken = splitToken[1]
	// Find user id in JWT token
	token, err := extractTokenFromJWT(reqToken)
	if err != nil {
		response := HTTPResponse{
			Error: FieldErrors{
				FieldError{
					Field: "-",
					Error: err.Error(),
				},
			},
			Status: http.StatusBadRequest,
		}
		RespondJson(w, r, response)
		return
	}
	userToken, _ := readJWTClaims(token)

	var pet Pet
	if err := db.Where("slug = ?", petSlug).Where("user_id = ?", userToken.id).First(&pet).Error; err != nil {
		var status int
		if err == gorm.ErrRecordNotFound {
			status = http.StatusNotFound
		} else {
			status = http.StatusBadRequest
		}

		response := HTTPResponse{
			Data: nil,
			Error: FieldErrors{
				FieldError{
					Field: "-",
					Error: err.Error(),
				},
			},
			Status: status,
		}
		RespondJson(w, r, response)
		return
	}
	response := HTTPResponse{
		Data:   pet,
		Error:  nil,
		Status: http.StatusOK,
	}

	RespondJson(w, r, response)
}

// TODO: Add ephemeral token with name and slug
func GetPublicPetBySlug(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	petSlug := params["slug"]

	var pet Pet
	if err := db.Preload("QRCode").Where("slug = ?", petSlug).First(&pet).Error; err != nil {
		var status int
		if err == gorm.ErrRecordNotFound {
			status = http.StatusNotFound
		} else {
			status = http.StatusBadRequest
		}
		response := HTTPResponse{
			Data: nil,
			Error: FieldErrors{
				FieldError{
					Field: "-",
					Error: err.Error(),
				},
			},
			Status: status,
		}
		RespondJson(w, r, response)
		return
	}

	validToken, err := generateReportJWT(&pet)
	if err != nil {
		response := HTTPResponse{
			Data: nil,
			Error: FieldErrors{
				FieldError{
					Field: "-",
					Error: err.Error(),
				},
			},
			Status: http.StatusInternalServerError,
		}
		RespondJson(w, r, response)
		return
	}

	var data = struct {
		Token string `json:"token"`
		Pet   Pet    `json:"pet"`
	}{
		Token: validToken,
		Pet:   pet,
	}

	response := HTTPResponse{
		Data:   data,
		Error:  nil,
		Status: http.StatusOK,
	}
	RespondJson(w, r, response)
}

func GetPetByID(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	petID, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid pet ID", http.StatusBadRequest)
		return
	}

	var pet Pet

	// Fetch a pet by ID from the database
	db.First(&pet, petID)

	if pet.ID == 0 {
		response := HTTPResponse{
			Data: nil,
			Error: FieldErrors{
				FieldError{
					Field: "-",
					Error: err.Error(),
				},
			},
			Status: http.StatusNotFound,
		}
		RespondJson(w, r, response)

		return
	}

	response := HTTPResponse{
		Data:   pet,
		Error:  nil,
		Status: http.StatusOK,
	}
	RespondJson(w, r, response)
}

func DeletePet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	petID := params["slug"]

	if !isValidUUID(petID) {
		response := HTTPResponse{
			Error: FieldErrors{
				{
					Field: "slug",
					Error: "L'identifiant de votre animal semble invalide",
				},
			},
			Status: http.StatusBadRequest,
		}
		RespondJson(w, r, response)
		return
	}

	// Soft delete the pet record
	db.Where("slug = ?", petID).Delete(&Pet{})

	response := HTTPResponse{
		Data:   "L'enregistrement a bien été supprimé",
		Error:  nil,
		Status: http.StatusOK,
	}
	RespondJson(w, r, response)
}
