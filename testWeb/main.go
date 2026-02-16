package main

import (
	"fmt"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"janeauto/config"
	"janeauto/db"
	"janeauto/web"
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

	e.GET("/", web.HomeHandler)
	e.GET("/attest", web.AttestFormHandler)
	e.GET("/policies", web.PoliciesHandler)
	e.GET("/debug-jane", web.DebugJaneHandler)

	e.POST("/attest/run", web.AttestRunHandler)
	e.POST("/execute/:policyName", web.ExecutePolicyHandler)

	addr := fmt.Sprintf(":%d", config.ConfigData.Rest.Port)
	log.Fatal(e.Start(addr))
}
