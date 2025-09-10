package main 

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, `
    <!DOCTYPE html>
    <html>
    <head><title>Jane Auto Test</title></head>
    <body>
	<h1>press button</h1>
	<form action="/policies" method="get">
	    <button style="font-size:18px;padding:8px 16px;">Connected! Show POlicies</button>
	</form>
    </body>
    </html>
    `)
}

func policiesHandler(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    collection := client.Database("testdb").Collection("policies")
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

    fmt.Fprintln(w, "<!DOCTYPE html><html><head><title>Policies</title></head><body>")
    fmt.Fprintln(w, "<h1>All Policies</h1>")

    for _, policy := range policies {
	fmt.Fprintf(w, `
	    <h2>%s</h2>
	    <p><b>Description:</b> %s</p>
	    <p><b>Jane:</b> %s</p>
	    <h3>Collection</h3>
	    <p><b>Items:</b> %v</p>
	    <p><b>Tags:</b> %v</p>
	    <p><b>Names:</b> %v</p>
	    <form action="/execute" method="post">
		<input type="hidden" name="policy" value="%s">
		<button type="submit">Execute</button>
	    </form>
	    <hr>
	`,
	    policy.Name, 
	    policy.Description,
	    policy.Jane,
	    strings.Join(policy.Collection.Items, ", "),
	    strings.Join(policy.Collection.Tags, ", "),
	    strings.Join(policy.Collection.Names, ", "),
	    policy.Name,
	)
    }

    fmt.Fprintln(w, `<p><a href="/">Back to home</a></p>`)
    fmt.Fprintln(w, "</body></html>")
}

func executeHandler(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    policyName := r.FormValue("policy")

    var policy Policy
    collection := client.Database("testdb").Collection("policies")
    if err := collection.FindOne(ctx, map[string]interface{}{"name": policyName}).Decode(&policy); err != nil {
	http.Error(w, "Policy not found", http.StatusNotFound)
	return
    }

    elements, err := executePolicy(policy)
    if err != nil {
	http.Error(w, "Execution Failed: "+err.Error(), http.StatusInternalServerError)
	return
    }

    accept := r.Header.Get("Accept")
    if strings.Contains(accept, "json") {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
	    "policy": policy.Name,
	    "elements": elements,
	})
    } else {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<h1>Executed POlicy: %s</h1>", policy.Name)
	fmt.Fprintf(w, "<ul>")
	for _, e := range elements {
	    fmt.Fprintf(w, "<li>%s</li>", e)
	}
	fmt.Fprintf(w, "</ul>")
	fmt.Fprintf(w, `<p><a href="/policies">Back to policies</a></p>`)
      }
}
