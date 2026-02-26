package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ai-reading-assistant/internal/handler"
)

// Setup registers HTTP routes and returns a ready-to-run Gin engine.
func Setup(inspirationHandler *handler.InspirationHandler) *gin.Engine {
	r := gin.Default()

	r.Use(corsPolicy())

	sessionGroup := r.Group("/inspiration/session")
	{
		sessionGroup.POST("/create", inspirationHandler.CreateInspirationSession)
		sessionGroup.GET("/get", inspirationHandler.GetInspirationSession)
	}

	messageGroup := r.Group("/inspiration/chat")
	{
		messageGroup.POST("/completion", inspirationHandler.ChatCompletion)
	}

	inspirationGroup := r.Group("/inspiration/inspiration")
	{
		inspirationGroup.POST("/favorite", inspirationHandler.FavoriteInspiration)
		inspirationGroup.POST("/unfavorite", inspirationHandler.UnfavoriteInspiration)
	}

	return r
}

// corsPolicy provides a minimal CORS policy for the Vite dev server.
func corsPolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		const origin = "http://localhost:5173"

		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", origin)
		h.Set("Access-Control-Allow-Credentials", "true")
		h.Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		h.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
