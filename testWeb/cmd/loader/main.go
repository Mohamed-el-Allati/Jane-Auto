package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	//MongoDB connection
	uri := "mongodb://localhost:27017"
	dbName := "testdb"
	collectionName := "policies"

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer client.Disconnect(context.Background())

	collection := client.Database(dbName).Collection(collectionName)

	// Finds all JSON files in the examples folder. edit it for your folder
	files, err := filepath.Glob("policies/examples/*.json")
	if err != nil {
		log.Fatal("Error finding JSON files:", err)
	}

	if len(files) == 0 {
		fmt.Println("No policy files found in policies/examples/")
		return
	}

	for _, file := range files {
		fmt.Printf("Processing %s...\n", file)

		data, err := ioutil.ReadFile(file)
		if err != nil {
			log.Printf("Error reading %s: %v", file, err)
			continue
		}

		var policy map[string]interface{}
		if err := json.Unmarshal(data, &policy); err != nil {
			log.Printf("Error parsing %s: %v", file, err)
			continue
		}

		// Ensures the policy has a name field
		name, ok := policy["name"].(string)
		if !ok || name == "" {
			log.Printf("Skipping %s: missing or invalid 'name' field", file)
			continue
		}

		// updates file if it exists, creates new if not
		filter := map[string]interface{}{"name": name}
		update := map[string]interface{}{"$set": policy}
		opts := options.Update().SetUpsert(true)

		result, err := collection.UpdateOne(context.Background(), filter, update, opts)
		if err != nil {
			log.Printf("Error upserting %s: %v", name, err)
			continue
		}

		if result.UpsertedID != nil {
			fmt.Printf(" Inserted new policy: %s\n", name)
		} else if result.ModifiedCount > 0 {
			fmt.Printf(" Updated existing policy: %s\n", name)
		} else {
			fmt.Printf(" Policy unchanged: %s\n", name)
		}
	}

}
