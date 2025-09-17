package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"rfap_back/app/entities"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	Number string `json:"number"`
	jwt.RegisteredClaims
}

func Server(port string, db *mongo.Database) error {
	http.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"health": "OK"}`)
	})

	http.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error": "can't read request body. Invalid JSON."}`)
			return
		}

		if _, ok := body["password"]; !ok {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error": "Missing required field 'password'."}`)
			return
		}

		if _, ok := body["number"]; !ok {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error": "Missing required field 'number'."}`)
			return
		}

		col := db.Collection("user")

		filter := bson.D{{"number", body["number"]}}

		var user entities.User
		err := col.FindOne(context.TODO(), filter).Decode(&user)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, `{"error": "Invalid Number or Password"}`)
				return
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"error": "Failed to read from DB.", "errMsg": "%s"}`, err)
				return
			}
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.HashedPass), []byte(body["password"]))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error": "Invalid Number or Password"}`)
			return
		}

		key := os.Getenv("JWT_KEY")
		expirationTime := time.Now().Add(100000 * time.Minute)
		claims := &Claims{
			Number: user.Number,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expirationTime),
				Issuer:    "rfap_back",
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		tokenString, err := token.SignedString([]byte(key))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error": "Failed to sign token", "errMsg": "%s"}`, err)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"token": %s}`, tokenString)
		return
	})

	http.HandleFunc("POST /signup", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error": "can't read request body. Invalid JSON."}`)
			return
		}

		if _, ok := body["password"]; !ok {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error": "Missing required field 'password'."}`)
			return
		}

		if _, ok := body["number"]; !ok {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error": "Missing required field 'number'."}`)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(body["password"]), bcrypt.DefaultCost)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error": "Failed to hash password.", "errMsg": "%s"}`, err)
			return
		}

		col := db.Collection("user")

		isNumberDuplicated, err := isUserNumberDuplicated(col, body["number"])
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error": "Failed to query database for duplicated number", "errMsg": "%s"}`, err)
			return
		}

		if isNumberDuplicated {
			w.WriteHeader(http.StatusConflict)
			fmt.Fprintf(w, `{"error": "Number '%s' already registered in the database"}`, body["number"])
			return
		}

		user := entities.User{
			Number:     body["number"],
			HashedPass: string(hash),
		}

		result, err := col.InsertOne(context.TODO(), user)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error": "Failed to insert user data.", "errMsg": "%s"}`, err)
			return
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"insertedId": "%s"}`, result.InsertedID)
	})

	log.Printf("Starting server on port: %s\n", port)
	return http.ListenAndServe(port, nil)
}

func isUserNumberDuplicated(coll *mongo.Collection, number string) (bool, error) {
	filter := bson.D{{"number", number}}

	var user entities.User
	err := coll.FindOne(context.TODO(), filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}
