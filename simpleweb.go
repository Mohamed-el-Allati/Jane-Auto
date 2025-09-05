package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "time"
    "strings"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/mongo/readpref"
)

type Policy struct {
    Name	string		 `bson:"name" json:"name"`
    Description	string		 `bson:"description" json:"description"`
    Jane	string		 `bson:"jane" json:"jane"`
    Collection	PolicyCollection `bson:"collection" json:"collection"`
}

type PolicyCollection struct {
    Items []string `bson:"items" json:"items"`
    Tags  []string `bson:"tags" json:"tags"`
    Names []string `bson:"names" json:"names"` 
}

var client *mongo.Client

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var err error
    client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://172.16.222.58:27017/"))
    if err != nil {
	log.Fatal(err)
    }

    if err := client.Ping(ctx, readpref.Primary()); err != nil {
	log.Fatal("Cannot Establish Connection to MongoDB:", err)
    }
    fmt.Println("Connected to MongoDB!")

    collection := client.Database("testdb").Collection("policies")
    testPolicy := Policy{
	Name:		"testAttest",
	Description:	"This attests all the x32 servers",
	Jane:		"http://127.0.0.1:8540",
	Collection: PolicyCollection{
	    Tags:  []string{"windows", "x32"},
	    Names: []string{"mohamed", "bobafet"},
	},
    }
    _, err = collection.InsertOne(ctx, testPolicy)
    if err != nil {
	fmt.Println("Insert might have failed, check for duplicates:", err)
    } else {
	fmt.Println("Inserted sample policy successfully")
    }

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var policy Policy 
	err := collection.FindOne(ctx, map[string]interface{}{"name": "BigAttest"}).Decode(&policy)
	if err != nil {
            http.Error(w, "Policy not found", http.StatusNotFound)
	    return
    	}
	
	fmt.Fprintf(w, `
	<!DOCTYPE html>
	<html>
	<head> <title>MongoDB Connection</title></head>
	<body>
	    <h1>%s</h1>
	    <p><b>Description:</b> %s</p>
	    <p><b>Jane:</b> %s</p>
	    <h2>Collection</h2>
	    <p><b>Items:</b> %v</p>
	    <p><b>Tags:</b> %v</p>
	    <p><b>Names:</b> %v</p>
	    <br>
	    <button style="font-size:20px;padding:10px 20px;">Connected to MongoDB!</button>
	</body>
	</html>
	`, policy.Name, 
	   policy.Description, 
	   policy.Jane, 
	   strings.Join(policy.Collection.Items, ", "), 
	   strings.Join(policy.Collection.Tags, ", "), 
	   strings.Join(policy.Collection.Names, ", "),
	)
    })
    fmt.Println("Test script simpleweb.go")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
