package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/gin-gonic/gin"
)

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(c *gin.Context, err error) {
	var de *domain.Error
	if errors.As(err, &de) {
		c.JSON(statusFor(de.Code), errorEnvelope{Error: errorBody{Code: de.Code, Message: de.Message}})
		return
	}
	slog.Error("handler",
		"err", err,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
	)
	c.JSON(http.StatusInternalServerError, errorEnvelope{Error: errorBody{
		Code:    domain.CodeInvalidBody,
		Message: "Internal server error.",
	}})
}

func statusFor(code string) int {
	switch code {
	case domain.CodeNotFound, domain.CodeSessionNotActive:
		return http.StatusNotFound
	case domain.CodeInvalidBody, domain.CodeInvalidQuery, domain.CodeInvalidJSON:
		return http.StatusBadRequest
	case domain.CodeSessionAlreadyActive, domain.CodeProjectArchived, domain.CodeCodeInUse,
		domain.CodeNameInUse, domain.CodeLastActiveProject, domain.CodeProjectHasSessions,
		domain.CodeActivityTypeHasSessions, domain.CodeInvalidTransition, domain.CodeUsernameInUse:
		return http.StatusConflict
	case domain.CodeUnauthorized, domain.CodeInvalidCredentials:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

func decodeJSON(c *gin.Context, dest any) error {
	dec := json.NewDecoder(c.Request.Body)
	if err := dec.Decode(dest); err != nil {
		if errors.Is(err, io.EOF) {
			return domain.ErrInvalidJSON()
		}
		var syn *json.SyntaxError
		var typ *json.UnmarshalTypeError
		if errors.As(err, &syn) || errors.As(err, &typ) {
			return domain.ErrInvalidJSON()
		}
		return domain.ErrInvalidBody(err.Error())
	}
	return nil
}
