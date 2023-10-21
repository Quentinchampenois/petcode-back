package main

import (
	"encoding/base64"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
	"net/http"
	"strings"
)

type QRCode struct {
	gorm.Model
	Url    string `json:"url"`
	Base64 string `json:"base64"`
	PetID  uint   `json:"pet_id"`
}

func GenerateBase64(url string) string {
	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		fmt.Println("Could not generate QR code:", err)
		return ""
	}

	return base64.StdEncoding.EncodeToString(png)
}

func GetPetQRCode(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	petSlug := params["slug"]
	var pet Pet
	reqToken := r.Header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer ")
	reqToken = splitToken[1]
	token, err := extractTokenFromJWT(reqToken)
	if err != nil {
		response := HTTPResponse{
			Error:  err.Error(),
			Status: http.StatusBadRequest,
		}
		RespondJson(w, r, response)
		return
	}
	userToken, _ := readJWTClaims(token)

	db.Preload("QRCode").Where("user_id = ?", userToken.id).Where("slug = ?", petSlug).Find(&pet)

	response := HTTPResponse{
		Data:   pet,
		Error:  nil,
		Status: http.StatusOK,
	}

	RespondJson(w, r, response)
}
