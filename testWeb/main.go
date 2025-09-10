package main

import (
    "fmt"
    "log"
    "net/http"
)

func main() {
    connectDB("mongodb://172.16.222.58:27017")

    http.HandleFunc("/", homeHandler)
    http.HandleFunc("/policies", policiesHandler)
    http.HandleFunc("/execute", executeHandler)

    fmt.Println("Server is running at http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
