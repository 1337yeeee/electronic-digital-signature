package routes

import (
	"electronic-digital-signature/internal/app/container"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRouter(container *container.AppContainer) *gin.Engine {
	//TODO routes
	r := gin.Default()

	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	//api := r.Group("/api")

	return r
}
