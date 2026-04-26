package routes

import (
	"net/http"

	"electronic-digital-signature/internal/app/container"

	"github.com/gin-gonic/gin"
)

func SetupRouter(appContainer *container.AppContainer) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"status": "ok",
			},
		})
	})

	api := r.Group("/api/v1")
	if appContainer == nil {
		api.GET("/server/public-key", handlerNotConfigured)
		api.POST("/server/messages", handlerNotConfigured)
		api.GET("/server/messages/:id", handlerNotConfigured)
		api.POST("/signatures/verify", handlerNotConfigured)
		api.POST("/documents", handlerNotConfigured)
		api.POST("/documents/:id/send", handlerNotConfigured)
		api.POST("/documents/verify-decrypt", handlerNotConfigured)
		api.POST("/users/register", handlerNotConfigured)
		api.GET("/users/:id", handlerNotConfigured)
		api.POST("/auth/login", handlerNotConfigured)
		api.GET("/auth/me", handlerNotConfigured)
		return r
	}

	if appContainer.SignatureHandler == nil {
		api.GET("/server/public-key", handlerNotConfigured)
		api.POST("/server/messages", handlerNotConfigured)
		api.GET("/server/messages/:id", handlerNotConfigured)
		api.POST("/signatures/verify", handlerNotConfigured)
	} else {
		signatureHandler := appContainer.SignatureHandler
		api.GET("/server/public-key", signatureHandler.GetServerPublicKey)
		if appContainer.AuthMiddleware == nil {
			api.POST("/server/messages", handlerNotConfigured)
		} else {
			api.POST("/server/messages", appContainer.AuthMiddleware.RequireAuth(), signatureHandler.IssueServerMessage)
		}
		api.GET("/server/messages/:id", signatureHandler.GetServerMessage)
		api.POST("/signatures/verify", signatureHandler.VerifyClientSignature)
	}

	if appContainer.DocumentHandler == nil {
		api.POST("/documents", handlerNotConfigured)
		api.POST("/documents/:id/send", handlerNotConfigured)
		api.POST("/documents/verify-decrypt", handlerNotConfigured)
	} else {
		if appContainer.AuthMiddleware == nil {
			api.POST("/documents", handlerNotConfigured)
			api.POST("/documents/:id/send", handlerNotConfigured)
		} else {
			api.POST("/documents", appContainer.AuthMiddleware.RequireAuth(), appContainer.DocumentHandler.UploadDocument)
			api.POST("/documents/:id/send", appContainer.AuthMiddleware.RequireAuth(), appContainer.DocumentHandler.SendDocument)
		}
		api.POST("/documents/verify-decrypt", appContainer.DocumentHandler.VerifyDecryptPackage)
	}

	if appContainer.UserHandler == nil {
		api.POST("/users/register", handlerNotConfigured)
		api.GET("/users/:id", handlerNotConfigured)
	} else {
		api.POST("/users/register", appContainer.UserHandler.Register)
		api.GET("/users/:id", appContainer.UserHandler.GetByID)
	}

	if appContainer.AuthHandler == nil {
		api.POST("/auth/login", handlerNotConfigured)
		api.GET("/auth/me", handlerNotConfigured)
	} else {
		api.POST("/auth/login", appContainer.AuthHandler.Login)
		if appContainer.AuthMiddleware == nil {
			api.GET("/auth/me", handlerNotConfigured)
		} else {
			api.GET("/auth/me", appContainer.AuthMiddleware.RequireAuth(), appContainer.AuthHandler.Me)
		}
	}

	return r
}

func handlerNotConfigured(ctx *gin.Context) {
	ctx.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "internal_error",
			"message": "Requested handler is not configured.",
		},
	})
}
