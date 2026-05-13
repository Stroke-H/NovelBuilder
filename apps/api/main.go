package main

import (
	"log"
	"net/http"
	"os"

	"novel-generater-api/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.SetTrustedProxies(nil)

	config := cors.DefaultConfig()
	config.AllowOriginFunc = func(origin string) bool { return true }
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Accept"}
	config.AllowCredentials = true
	r.Use(cors.New(config))

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", services.RegisterHandler)
			auth.POST("/login", services.LoginHandler)
			auth.POST("/logout", services.LogoutHandler)
			auth.GET("/me", services.GetUserMeHandler)
			auth.PUT("/profile", services.UpdateCurrentUserProfileHandler)
		}

		databaseGroup := api.Group("/database")
		{
			databaseGroup.GET("/config", services.GetDatabaseConfigHandler)
			databaseGroup.GET("/health", services.GetDatabaseHealthHandler)
		}

		services.RegisterNovelWriterRoutes(api)
	}

	host := os.Getenv("NOVEL_GENERATER_BACKEND_HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	port := os.Getenv("NOVEL_GENERATER_BACKEND_PORT")
	if port == "" {
		port = "19081"
	}
	addr := host + ":" + port

	log.Println("NovelGenerater API starting on " + addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
