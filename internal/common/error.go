package common

import (
	"errors"
	"fmt"
	"net/http"

	"inforadar/internal/constants"
)

// AppError keeps the HTTP status separate from the stable business code.
type AppError struct {
	HTTPCode int
	Code     int
	Message  string
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Is(target error) bool {
	var other *AppError
	return errors.As(target, &other) && e.Code == other.Code
}

func NewError(httpCode, code int, message string) *AppError {
	return &AppError{HTTPCode: httpCode, Code: code, Message: message}
}

func (e *AppError) WithMessage(message string) *AppError {
	return NewError(e.HTTPCode, e.Code, message)
}

var (
	ErrParams                   = NewError(http.StatusBadRequest, constants.CodeParams, constants.MessageParams)
	ErrInvalidInput             = NewError(http.StatusBadRequest, constants.CodeInvalidInput, constants.MessageInvalidInput)
	ErrInvalidToken             = NewError(http.StatusBadRequest, constants.CodeInvalidToken, constants.MessageInvalidToken)
	ErrCurrentPassword          = NewError(http.StatusBadRequest, constants.CodeCurrentPassword, constants.MessageCurrentPassword)
	ErrSamePassword             = NewError(http.StatusBadRequest, constants.CodeSamePassword, constants.MessageSamePassword)
	ErrNotLogin                 = NewError(http.StatusUnauthorized, constants.CodeNotLogin, constants.MessageNotLogin)
	ErrInvalidCredentials       = NewError(http.StatusUnauthorized, constants.CodeInvalidCredentials, constants.MessageInvalidCredentials)
	ErrForbidden                = NewError(http.StatusForbidden, constants.CodeForbidden, constants.MessageForbidden)
	ErrOriginNotAllowed         = NewError(http.StatusForbidden, constants.CodeOriginNotAllowed, constants.MessageOriginNotAllowed)
	ErrOriginRequired           = NewError(http.StatusForbidden, constants.CodeOriginRequired, constants.MessageOriginRequired)
	ErrNotFound                 = NewError(http.StatusNotFound, constants.CodeNotFound, constants.MessageNotFound)
	ErrDuplicateEmail           = NewError(http.StatusConflict, constants.CodeDuplicateEmail, constants.MessageDuplicateEmail)
	ErrTooManyRequests          = NewError(http.StatusTooManyRequests, constants.CodeTooManyRequests, constants.MessageTooManyRequests)
	ErrRequestTooLarge          = NewError(http.StatusRequestEntityTooLarge, constants.CodeRequestTooLarge, constants.MessageRequestTooLarge)
	ErrSystem                   = NewError(http.StatusInternalServerError, constants.CodeSystem, constants.MessageSystem)
	ErrMailUnavailable          = NewError(http.StatusServiceUnavailable, constants.CodeMailUnavailable, constants.MessageMailUnavailable)
	ErrPasswordResetUnavailable = NewError(http.StatusServiceUnavailable, constants.CodePasswordResetUnavailable, constants.MessagePasswordResetUnavailable)
	ErrRadarUnconfigured        = NewError(http.StatusBadRequest, constants.CodeRadarUnconfigured, constants.MessageRadarUnconfigured)
	ErrRadarBusy                = NewError(http.StatusConflict, constants.CodeRadarBusy, constants.MessageRadarBusy)
	ErrRadarCooldown            = NewError(http.StatusTooManyRequests, constants.CodeRadarCooldown, constants.MessageRadarCooldown)
	ErrProviderUnavailable      = NewError(http.StatusServiceUnavailable, constants.CodeProviderUnavailable, constants.MessageProviderUnavailable)
)
