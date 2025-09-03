package main

import (
    "fmt"
    "log"
    "context"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/mongo/readpref"
)

func main() {
    uri := "mongodb://172.16.222.58:27017/"
    clientOptions := options.Client().ApplyURI(uri)
    client, err := mongo.Connect(context.TODO(), clientOptions)
    if err != nil {
	log.Fatal(err)
    }
    defer client.Disconnect(context.TODO())
    err = client.Ping(context.TODO(), readpref.Primary())
    if err != nil {
	log.Fatal(err)
    }
    fmt.Println("Connected to MongoDB")
}

