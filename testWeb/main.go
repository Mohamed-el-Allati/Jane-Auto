package main

import (
	"fmt"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"janeauto/config"
	"janeauto/db"
)

func main() {
	//connectDB("mongodb://172.16.222.58:27017")
	config.ParseFlags()
	config.SetupConfiguration()

	fmt.Println("Mongo URI:", config.ConfigData.Database.Connection)
	fmt.Println("JANE URL:", config.ConfigData.Jane.URL)

	db.Connect(config.ConfigData.Database.Connection)

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/", homeHandler)
	e.GET("/attest", attestFormHandler)
	e.GET("/policies", policiesHandler)
	e.GET("/debug-jane", debugJaneHandler)

	e.POST("/attest/run", attestRunHandler)
	e.POST("/execute/:policyName", executePolicyHandler)

	addr := fmt.Sprintf(":%d", config.ConfigData.Rest.Port)
	log.Fatal(e.Start(addr))
}
