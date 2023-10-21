_WIP: Just a personal project with no-predefined roadmap_
# Petcode API backend

Functional Petcode API server using Gorm and Gorilla Mux. 

## Getting Started

Expose the backend server using Docker

```bash
docker-compose up -d
```

## 👋 Usage

See an example of cURL request, a Postman collection is available but not published yet.

### Create a new pet

```bash
curl --location 'http://localhost:8080/pets' \
--header 'Content-Type: application/json' \
--data '{
    "name": "Croquette",
    "breed": "Malinois",
    "sexe": 0,
    "birthdate": "2014-07-31"

}'
```

## 💡 Functionalities

* CRUD Pet
* CRUD User
* JWT Authentication
* Logger middleware using Zerolog
* Gorm implementation
* Request UUID middleware
* Gorilla Mux implementation
* Docker containerization
* Qrcode Generation
* Postgresql database