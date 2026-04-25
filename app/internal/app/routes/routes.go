package routes

import (
	"net/http"

	"electronic-digital-signature/internal/app/container"

	"github.com/gin-gonic/gin"
)

func SetupRouter(appContainer *container.AppContainer) *gin.Engine {
	//TODO routes
	r := gin.Default()

	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	if appContainer == nil || appContainer.SignatureHandler == nil {
		api.GET("/server/public-key", handlerNotConfigured)
		api.POST("/server/messages", handlerNotConfigured)
		api.GET("/server/messages/:id", handlerNotConfigured)
		api.POST("/signatures/verify", handlerNotConfigured)
		return r
	}

	signatureHandler := appContainer.SignatureHandler
	api.GET("/server/public-key", signatureHandler.GetServerPublicKey)
	api.POST("/server/messages", signatureHandler.IssueServerMessage)
	api.GET("/server/messages/:id", signatureHandler.GetServerMessage)
	api.POST("/signatures/verify", signatureHandler.VerifyClientSignature)

	return r
}

func handlerNotConfigured(ctx *gin.Context) {
	ctx.JSON(http.StatusInternalServerError, gin.H{
		"error": "signature handler is not configured",
	})
}
