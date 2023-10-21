package main

import (
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
)

type User struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Email     string `gorm:"type:varchar(50);unique" json:"email"`
	Password  string `gorm:"type:varchar(100)" json:"password"`
	Name      string `gorm:"type:varchar(40)" json:"name"`
	Firstname string `gorm:"type:varchar(25)" json:"firstname"`
}

type UserRes struct {
	ID        uint   `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Firstname string `json:"firstname"`
}

func (u *User) validPassword(hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(hash))
	return err == nil
}

func signIn(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		LogErr(r, err)
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

	var foundUser User
	err = db.Where("email = ?", user.Email).First(&foundUser).Error
	if err != nil || foundUser.Email == "" {
		LogDebug(r, err.Error())
		response := HTTPResponse{
			Error: FieldErrors{
				FieldError{
					Field: "-",
					Error: "Veuillez vérifier vos identifiants",
				},
			},
			Status: http.StatusBadRequest,
		}
		RespondJson(w, r, response)
		return
	}

	if check := foundUser.validPassword(user.Password); !check {
		LogDebug(r, "Wrong password")
		response := HTTPResponse{
			Error: FieldErrors{
				FieldError{
					Field: "-",
					Error: "Veuillez vérifier vos identifiants",
				},
			},
			Status: http.StatusBadRequest,
		}
		RespondJson(w, r, response)
		return
	}

	validToken, err := generateJWT(&foundUser)
	if err != nil {
		LogErr(r, err)
		response := HTTPResponse{
			Error: FieldErrors{
				FieldError{
					Field: "-",
					Error: "Erreur serveur lors de la génération de votre jeton d'authentification. Veuillez réessayer plus tard",
				},
			},
			Status: http.StatusInternalServerError,
		}
		RespondJson(w, r, response)
		return
	}

	token := Token{
		Email:       foundUser.Email,
		TokenString: validToken,
	}

	response := HTTPResponse{
		Data:   token,
		Error:  nil,
		Status: http.StatusOK,
	}

	RespondJson(w, r, response)
}

func signUp(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
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

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
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

	user.Password = string(hash)
	// Create user in database
	err = db.Create(&user).Error
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

	validToken, err := generateJWT(&user)
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

	token := Token{
		TokenString: validToken,
	}

	response := HTTPResponse{
		Data:   token,
		Error:  nil,
		Status: http.StatusOK,
	}

	RespondJson(w, r, response)
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	reqToken := r.Header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer ")
	reqToken = splitToken[1]

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

	// Find user in database
	var user User
	err = db.Find(&user, userToken.id).Error
	if err != nil {
		response := HTTPResponse{
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

	userRes := UserRes{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Firstname: user.Firstname,
	}

	response := HTTPResponse{
		Data:   userRes,
		Error:  nil,
		Status: http.StatusOK,
	}

	RespondJson(w, r, response)
}
