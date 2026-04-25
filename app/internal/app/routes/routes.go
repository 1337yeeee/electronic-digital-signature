package routes

import (
	"net/http"

	"electronic-digital-signature/internal/app/container"
	"electronic-digital-signature/internal/app/dto"
	"electronic-digital-signature/internal/infra/crypto"

	"github.com/gin-gonic/gin"
)

func SetupRouter(appContainer *container.AppContainer) *gin.Engine {
	//TODO routes
	r := gin.Default()

	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	api.GET("/server/public-key", func(ctx *gin.Context) {
		if appContainer == nil || len(appContainer.ServerKeys.PublicKey) == 0 {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "server public key is not loaded",
			})
			return
		}

		ctx.JSON(http.StatusOK, dto.ServerPublicKeyResponse{
			Algorithm:    crypto.ECDSASHA256Algorithm,
			PublicKeyPEM: string(appContainer.ServerKeys.PublicKey),
		})
	})

	return r
}
