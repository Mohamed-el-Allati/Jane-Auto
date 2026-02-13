package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"janeauto/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var client *mongo.Client

// establishes connection to mongodb
func Connect(uri string) *mongo.Client {
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

// retrieves a single policy by its name
func GetPolicyByName(name string) (*models.Policy, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var policy models.Policy
	err := client.Database("testdb").Collection("policies").
		FindOne(ctx, bson.M{"name": name}).
		Decode(&policy)

	if err != nil {
		return nil, err
	}
	return &policy, nil
}

// Retrieves all policies from the database
func GetAllPolicies() ([]models.Policy, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := client.Database("testdb").Collection("policies").
		Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var policies []models.Policy
	if err = cursor.All(ctx, &policies); err != nil {
		return nil, err
	}
	return policies, nil
}
