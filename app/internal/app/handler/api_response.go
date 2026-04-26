package handler

import (
	"log"
	"strings"

	"electronic-digital-signature/internal/app/dto"

	"github.com/gin-gonic/gin"
)

func respondSuccess(ctx *gin.Context, status int, data any) {
	ctx.JSON(status, dto.SuccessResponse{
		Success: true,
		Data:    data,
	})
}

func respondError(ctx *gin.Context, status int, code, message string) {
	ctx.JSON(status, dto.ErrorResponse{
		Success: false,
		Error: dto.ErrorDetails{
			Code:    code,
			Message: message,
		},
	})
}

func logRequestError(ctx *gin.Context, scope string, err error) {
	if err == nil {
		return
	}

	log.Printf("%s %s %s user_id=%s: %v", scope, ctx.Request.Method, ctx.FullPath(), requestUserID(ctx), err)
}

func logRequestInfo(ctx *gin.Context, scope, message string) {
	log.Printf("%s %s %s user_id=%s: %s", scope, ctx.Request.Method, ctx.FullPath(), requestUserID(ctx), message)
}

func requestUserID(ctx *gin.Context) string {
	if ctx == nil {
		return "anonymous"
	}
	currentUser, ok := currentUserFromContext(ctx)
	if !ok || currentUser == nil || strings.TrimSpace(currentUser.ID) == "" {
		return "anonymous"
	}

	return currentUser.ID
}
