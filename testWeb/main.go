package main

import (
    "fmt"
    "log"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"

    "../config"
)

func main() {
    //connectDB("mongodb://172.16.222.58:27017")
    config.ParseFlags()
    config.SetupConfiguration()

    fmt.Println("Mongo URI:", config.ConfigData.Database.Connection)
    fmt.Println("JANE URL:", config.ConfigData.Rest.Port)

    connectDB(config.ConfigData.Database.Connection)

    e := echo.New()

    e.Use(middleware.Logger())
    e.Use(middleware.Recover())

    e.GET("/", homeHandler)
    e.GET("/policies", policiesHandler)

    e.POST("/execute/:policyName", executePolicyHandler)

    addr := fmt.Sprintf(":%s", config.ConfigData.Rest.Port)
    log.Fatal(e.Start(addr))
}
