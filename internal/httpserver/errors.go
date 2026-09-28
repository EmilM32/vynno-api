package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

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
		if de.Code == domain.CodeRateLimited {
			c.Header("Retry-After", strconv.Itoa(retryAfterSeconds(de.RetryAfter)))
		}
		c.JSON(statusFor(de.Code), errorEnvelope{Error: errorBody{Code: de.Code, Message: de.Message}})
		return
	}
	slog.Error("handler",
		"err", err,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"request_id", requestIDFrom(c),
	)
	c.JSON(http.StatusInternalServerError, errorEnvelope{Error: errorBody{
		Code:    domain.CodeInternalError,
		Message: "Internal server error.",
	}})
}

func retryAfterSeconds(d time.Duration) int {
	if d <= 0 {
		return 1
	}
	secs := int(d / time.Second)
	if d%time.Second != 0 {
		secs++
	}
	if secs < 1 {
		return 1
	}
	return secs
}

func statusFor(code string) int {
	switch code {
	case domain.CodeNotFound, domain.CodeSessionNotActive:
		return http.StatusNotFound
	case domain.CodeInvalidBody, domain.CodeInvalidQuery, domain.CodeInvalidJSON:
		return http.StatusBadRequest
	case domain.CodeSessionAlreadyActive, domain.CodeProjectArchived, domain.CodeCodeInUse,
		domain.CodeNameInUse, domain.CodeLastActiveProject, domain.CodeProjectHasSessions,
		domain.CodeActivityTypeHasSessions, domain.CodeInvalidTransition, domain.CodeEmailInUse:
		return http.StatusConflict
	case domain.CodeUnauthorized, domain.CodeInvalidCredentials, domain.CodeInvalidCode:
		return http.StatusUnauthorized
	case domain.CodeRateLimited:
		return http.StatusTooManyRequests
	case domain.CodeInternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

func decodeJSON(c *gin.Context, dest any) error {
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return domain.ErrInvalidJSON()
		}
		var de *domain.Error
		if errors.As(err, &de) {
			return de
		}
		var syn *json.SyntaxError
		if errors.As(err, &syn) {
			return domain.ErrInvalidJSON()
		}
		var typ *json.UnmarshalTypeError
		if errors.As(err, &typ) {
			field := typ.Field
			if i := strings.LastIndex(field, "."); i >= 0 {
				field = field[i+1:]
			}
			if field == "" {
				field = "value"
			}
			return domain.ErrInvalidBody(field + " has the wrong type.")
		}
		if strings.Contains(err.Error(), "unknown field") {
			return domain.ErrInvalidBody("Unknown field.")
		}
		return domain.ErrInvalidBody(err.Error())
	}
	if dec.More() {
		return domain.ErrInvalidJSON()
	}
	return nil
}
