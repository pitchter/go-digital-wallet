package handler

import (
	"errors"
	"net/http"
	"strconv"

	"go-digital-wallet/internal/apperror"
	"go-digital-wallet/internal/dto"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func bindJSON[T any](c *gin.Context) (T, bool) {
	var request T
	if err := c.ShouldBindJSON(&request); err != nil {
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			details := map[string]string{}
			for _, validationErr := range validationErrs {
				details[validationErr.Field()] = validationErr.Tag()
			}

			writeError(c, apperror.Validation("one or more fields are invalid").WithDetails(details))
			return request, false
		}

		writeError(c, apperror.BadRequest("request payload is invalid"))
		return request, false
	}

	return request, true
}

func parseUUIDParam(c *gin.Context, key string) (uuid.UUID, bool) {
	value := c.Param(key)
	id, err := uuid.Parse(value)
	if err != nil {
		writeError(c, apperror.BadRequest("request payload is invalid"))
		return uuid.Nil, false
	}

	return id, true
}

func parsePagination(c *gin.Context) (int, int) {
	page := parseQueryInt(c, "page", 1)
	limit := parseQueryInt(c, "limit", 20)
	return page, limit
}

func parseQueryInt(c *gin.Context, key string, fallback int) int {
	value := c.Query(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func writeJSON(c *gin.Context, statusCode int, payload any) {
	c.JSON(statusCode, payload)
}

func writeError(c *gin.Context, err error) {
	appErr, ok := apperror.As(err)
	if !ok {
		appErr = apperror.Internal(err)
	}

	payload := dto.ErrorResponse{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
	}
	c.AbortWithStatusJSON(appErr.StatusCode, payload)
}

func noContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
