package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/mongo/readpref"
)

var client *mongo.Client

func connectDB(uri string) *mongo.Client {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    c, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
    if err != nil {
	log.Fatal(err)
    }

    if err := c.Ping(ctx, readpref.Primary()); err != nil {
	log.Fatal("Cannot connect to MongoDB:", err)
    }

    fmt.Println("Connected to MongoDB!")
    client = c
    return client
}
