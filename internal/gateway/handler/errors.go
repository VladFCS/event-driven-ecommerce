package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
)

func writeError(c *gin.Context, err error) {
	statusCode := statusCodeForError(err)
	slog.ErrorContext(
		c.Request.Context(),
		"http request failed",
		slog.String("method", c.Request.Method),
		slog.String("path", c.FullPath()),
		slog.Int("status_code", statusCode),
		slog.String("error", err.Error()),
	)

	c.JSON(statusCode, errorResponse{
		Error: publicErrorMessage(err),
	})
}

func writeBindError(c *gin.Context, err error, requestShape any, fallbackMessage string) {
	slog.WarnContext(
		c.Request.Context(),
		"http request binding failed",
		slog.String("method", c.Request.Method),
		slog.String("path", c.FullPath()),
		slog.String("error", err.Error()),
	)

	c.JSON(http.StatusBadRequest, errorResponse{
		Error: bindErrorMessage(err, requestShape, fallbackMessage),
	})
}

func statusCodeForError(err error) int {
	statusCode := http.StatusInternalServerError
	switch {
	case errors.Is(err, gatewayservice.ErrTimeout):
		statusCode = http.StatusGatewayTimeout
	case errors.Is(err, gatewayservice.ErrRequestCanceled):
		statusCode = http.StatusRequestTimeout
	case errors.Is(err, gatewayservice.ErrInvalidInput),
		errors.Is(err, gatewayservice.ErrIdempotencyConflict),
		errors.Is(err, gatewayservice.ErrPreconditionFailed),
		errors.Is(err, gatewayservice.ErrUnsupportedCurrency),
		errors.Is(err, gatewayservice.ErrUnsupportedPaymentMethod):
		statusCode = http.StatusBadRequest
	case errors.Is(err, gatewayservice.ErrDownstreamNotFound):
		statusCode = http.StatusNotFound
	case errors.Is(err, gatewayservice.ErrDownstreamFailed):
		statusCode = http.StatusBadGateway
	}

	if errors.Is(err, gatewayservice.ErrIdempotencyConflict) {
		statusCode = http.StatusConflict
	}

	return statusCode
}

type errorResponse struct {
	Error string `json:"error"`
}

func publicErrorMessage(err error) string {
	switch {
	case errors.Is(err, gatewayservice.ErrTimeout):
		return "request timed out"
	case errors.Is(err, gatewayservice.ErrRequestCanceled):
		return "request was canceled"
	case errors.Is(err, gatewayservice.ErrIdempotencyConflict):
		return "request conflicts with an existing idempotency key"
	case errors.Is(err, gatewayservice.ErrPreconditionFailed):
		return "operation cannot be performed in current resource state"
	case errors.Is(err, gatewayservice.ErrUnsupportedCurrency):
		return "unsupported currency"
	case errors.Is(err, gatewayservice.ErrUnsupportedPaymentMethod):
		return "unsupported payment method"
	case errors.Is(err, gatewayservice.ErrInvalidInput):
		return "invalid request parameters"
	case errors.Is(err, gatewayservice.ErrDownstreamNotFound):
		return "resource not found"
	case errors.Is(err, gatewayservice.ErrDownstreamFailed):
		return "upstream service unavailable"
	default:
		return "internal server error"
	}
}

func bindErrorMessage(err error, requestShape any, fallbackMessage string) string {
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return "invalid request body"
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return "invalid request body"
	}

	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) && len(validationErrs) > 0 {
		fieldName := publicFieldName(requestShape, validationErrs[0].StructField())
		switch validationErrs[0].Tag() {
		case "required":
			return fieldName + " is required"
		case "gt":
			return fieldName + " must be greater than 0"
		case "min":
			return fieldName + " is invalid"
		default:
			return fieldName + " is invalid"
		}
	}

	return fallbackMessage
}

func publicFieldName(requestShape any, structField string) string {
	t := reflect.TypeOf(requestShape)
	if t == nil {
		return strings.ToLower(structField)
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return strings.ToLower(structField)
	}

	field, ok := t.FieldByName(structField)
	if !ok {
		return strings.ToLower(structField)
	}

	for _, tagName := range []string{"json", "uri", "form"} {
		tagValue := field.Tag.Get(tagName)
		if tagValue == "" || tagValue == "-" {
			continue
		}

		name := strings.Split(tagValue, ",")[0]
		if name != "" {
			return name
		}
	}

	return strings.ToLower(structField)
}
