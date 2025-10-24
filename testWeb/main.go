package main

import (
    "fmt"
    "log"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

func main() {
    connectDB("mongodb://172.16.222.58:27017")

    e := echo.New()

    e.Use(middleware.Logger())
    e.Use(middleware.Recover())

    e.GET("/", homeHandler)
    e.GET("/policies", policiesHandler)
    e.GET("/execute/:policyName", attestPolicyHandler)

    e.POST("/execute/:policyName", executePolicyHandler)

    fmt.Println("Server is running at http://localhost:8080")
    log.Fatal(e.Start(":8080"))
}
