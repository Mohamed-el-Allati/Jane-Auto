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
    Attestation map[string]AttestItem   `bson:"attestation" json:"attestation"`
}

type AttestItem struct {
    Endpoint string             `bson:"endpoint" json:"endpoint"`
    Rules    map[string]Rule    `bson:"rules" json:"rules"`
}

type Rule struct {
    RVariable string    `bson:"rvariable" json:"rvariable"`
    Parameter string    `bson:"parameter" json:"parameter"`
    Decision  string    `bson:"decision"  json:"decision"`
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

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, `
	<!DOCTYPE html>
	<html>
	<head><title>Jane Test Web</title></head>
	<body>
	    <h1>press button</h1>
	    <form action="/policies" method="get">
		<button style="font-size:18px;padding:8px 16px;">Connected! Show Policies</button>
	    </form>
	</body>
	</html>
        `)
    })


    http.HandleFunc("/policies", func(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

        cursor, err := collection.Find(ctx, map[string]interface{}{})
	if err != nil {
	    http.Error(w, "Error getting policies", http.StatusInternalServerError)
	    return
	}
	defer cursor.Close(ctx)

	var policies []Policy
	if err = cursor.All(ctx, &policies); err != nil {
	    http.Error(w, "Error decoding policies", http.StatusInternalServerError)
	    return
        }
	
	fmt.Fprintln(w, "<!DOCTYPE html><html><head><title><Policies></title></head><body>")
	fmt.Fprintln(w, "<h1>All printed Policies</h1>")
    	
	for _, policy := range policies {
	    fmt.Fprintf(w, `
	    	<h2>%s</h2>
	    	<p><b>Description:</b> %s</p>
	    	<p><b>Jane:</b> %s</p>
	    	<h2>Collection</h2>
	    	<p><b>Items:</b> %v</p>
	    	<p><b>Tags:</b> %v</p>
	    	<p><b>Names:</b> %v</p>
	    	<hr>
	    `,
	   	policy.Name, 
	  	policy.Description, 
	   	policy.Jane, 
	   	strings.Join(policy.Collection.Items, ", "), 
	   	strings.Join(policy.Collection.Tags, ", "), 
	   	strings.Join(policy.Collection.Names, ", "),
	    )
    	}

    	fmt.Fprintln(w, `<p><a href="/">Back to the main page</a></p>`)
	fmt.Fprintln(w, "</body></html>")
    })
    
    fmt.Println("Server running at http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
