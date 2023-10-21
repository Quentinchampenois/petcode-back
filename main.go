package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http"
	"os"
)

var db *gorm.DB

type HTTPResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Error  interface{} `json:"error,omitempty"`
	Status int         `json:"status"`
}

func (r *HTTPResponse) ToJsonBytes() []byte {
	res, _ := json.Marshal(r)
	return res
}

type FieldErrors []FieldError

func (fe FieldErrors) ToJsonBytes() ([]byte, error) {
	return json.Marshal(fe)
}

type FieldError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

func (fe *FieldError) ToJsonBytes() ([]byte, error) {
	return json.Marshal(fe)
}

type DatabaseConf struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

func initialize() {
	// Initialize the database connection using Gorm
	dbConf := DatabaseConf{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USERNAME"),
		Password: os.Getenv("DB_PASSWORD"),
		Name:     os.Getenv("DB_NAME"),
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Europe/paris",
		dbConf.Host, dbConf.Port, dbConf.User, dbConf.Password, dbConf.Name)

	fmt.Println("Postgresql connexion ... \n DSN : ", dsn)
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal().Msg(err.Error())
	} else {
		fmt.Println("Connexion established !")
	}

	if err := db.AutoMigrate(&User{}, &Pet{}, &QRCode{}, &Report{}); err != nil {
		log.Fatal().Msg(err.Error())
	}

	var users []User
	db.Find(&users)

	if os.Getenv("SEED") == "true" && len(users) < 2 {
		log.Info().Msg("Seeding database...")

		encryptedPassword, _ := encryptPassword("password")

		users := []User{
			{
				Email:     "john@doe.org",
				Password:  encryptedPassword,
				Name:      "Doe",
				Firstname: "John",
			},
			{
				Email:     "jane@doe.org",
				Password:  encryptedPassword,
				Name:      "Doe",
				Firstname: "Jane",
			},
		}

		db.Create(users)

		fmt.Println("Register 'Médor' the Labrador...")
		_ = db.Create(&Pet{
			Name:      "Médor",
			Breed:     "Labrador",
			Sexe:      "male",
			Birthdate: "01/01/2019",
			Slug:      (uuid.New()).String(),
			User:      users[0],
		})

		fmt.Println(len((uuid.New()).String()))
		fmt.Println("Register 'Pyla' the Beagle...")
		_ = db.Create(&Pet{
			Name:      "Pyla",
			Breed:     "Beagle",
			Sexe:      "female",
			Birthdate: "06/01/2020",
			Slug:      (uuid.New()).String(),
			User:      users[0],
		})

		fmt.Println("Register 'Brutus' the Caniche...")
		_ = db.Create(&Pet{
			Name:      "Brutus",
			Breed:     "Caniche",
			Sexe:      "male",
			Birthdate: "12/04/2022",
			Slug:      (uuid.New()).String(),
			User:      users[1],
		})

		fmt.Println("Register 'Pluto' the Yorkshire...")
		_ = db.Create(&Pet{
			Name:      "Pluto",
			Breed:     "Yorkshire",
			Sexe:      "male",
			Birthdate: "09/04/2023",
			Slug:      (uuid.New()).String(),
			User:      users[0],
		})
	}
}

func encryptPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	zerolog.TimeFieldFormat = "2006-01-02T15:04:05.999Z07:00"

	if err := godotenv.Load(".env"); err != nil {
		fmt.Println("No .env file found")
	}

	frontUrl := os.Getenv("FRONTEND_URL")

	initialize()

	// Create a CORS handler with the desired CORS options
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{frontUrl},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	// Create a new router
	router := mux.NewRouter()
	router.HandleFunc("/signin", signIn).Methods("POST")
	router.HandleFunc("/signup", signUp).Methods("POST")
	router.HandleFunc("/pet/{slug}", GetPublicPetBySlug).Methods("GET")
	router.HandleFunc("/pet/{slug}/report", CreateReport).Methods("POST")

	petsRouter := router.PathPrefix("/pets").Subrouter()

	// Define CRUD routes
	petsRouter.HandleFunc("", CreatePet).Methods("POST")
	petsRouter.HandleFunc("/", CreatePet).Methods("POST")
	petsRouter.HandleFunc("/", GetPets).Methods("GET")
	petsRouter.HandleFunc("", GetPets).Methods("GET")
	//petsRouter.HandleFunc("/{id}", GetPetByID).Methods("GET")
	petsRouter.HandleFunc("/{slug}", GetPetBySlug).Methods("GET")
	petsRouter.HandleFunc("/{slug}", UpdatePet).Methods("PUT")
	petsRouter.HandleFunc("/{slug}", DeletePet).Methods("DELETE")
	petsRouter.HandleFunc("/{slug}/qrcode", GetPetQRCode).Methods("GET")

	usersRouter := router.PathPrefix("/user").Subrouter()
	usersRouter.HandleFunc("/me", GetUser).Methods("GET")

	// Use the CORS handler as middleware for your app
	handler := c.Handler(router)
	router.Use(requestIDMiddleware)
	router.Use(zerologMiddleware)
	petsRouter.Use(isAuthorized)
	usersRouter.Use(isAuthorized)

	// Start the HTTP server
	http.Handle("/", router)
	fmt.Println("Server started on port 8080")
	log.Fatal().Msg(http.ListenAndServe(":8080", handler).Error())
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		// TODO: Fix CI warning SA1029
		ctx := context.WithValue(r.Context(), "requestID", requestID)

		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func zerologMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info().
			Str("RequestID", r.Context().Value("requestID").(string)).
			Str("Method", r.Method).
			Str("UserAgent", r.UserAgent()).
			Str("IP", r.RemoteAddr).
			Str("X-Forwarded-For", r.Header.Get("X-Forwarded-For")).
			Msg(r.URL.Path)

		next.ServeHTTP(w, r)
	})
}

func LogErr(r *http.Request, err error) {
	log.Error().
		Str("RequestID", r.Context().Value("requestID").(string)).
		Str("Method", r.Method).
		Str("UserAgent", r.UserAgent()).
		Str("IP", r.RemoteAddr).
		Str("X-Forwarded-For", r.Header.Get("X-Forwarded-For")).
		Str("Path", r.URL.Path).
		Msg(err.Error())
}
func LogDebug(r *http.Request, msg string) {
	log.Debug().
		Str("RequestID", r.Context().Value("requestID").(string)).
		Str("Method", r.Method).
		Str("IP", r.RemoteAddr).
		Str("X-Forwarded-For", r.Header.Get("X-Forwarded-For")).
		Str("Path", r.URL.Path).
		Msg(msg)
}
