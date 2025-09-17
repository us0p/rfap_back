package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"rfap_back/app/server"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(os.Getenv("MONGO_CONNECTION_STRING")).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(opts)
	if err != nil {
		log.Fatal(err)
	}

	db := client.Database("rfap")

	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			log.Fatal(err)
		}
	}()

	if err = client.Ping(context.TODO(), readpref.Primary()); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")

	err = server.Server(fmt.Sprintf(":%s", os.Getenv("PORT")), db)
	if err != nil {
		log.Fatal(err)
	}
}
