package routes

import (
	"electronic-digital-signature/internal/app/container"
	"github.com/gin-gonic/gin"
)

func SetupRouter(container *container.AppContainer) *gin.Engine {
	//TODO routes
	r := gin.Default()

	//api := r.Group("/api")

	return r
}
