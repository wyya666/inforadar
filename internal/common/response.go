package common

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"inforadar/internal/constants"
)

// BaseResponse is the JSON envelope shared by all API endpoints.
type BaseResponse struct {
	Code    int    `json:"code"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message"`
}

func Success(data any) BaseResponse {
	return BaseResponse{Code: constants.CodeSuccess, Data: data, Message: constants.MessageOK}
}

func SuccessWithMsg(data any, message string) BaseResponse {
	return BaseResponse{Code: constants.CodeSuccess, Data: data, Message: message}
}

func Error(err *AppError) BaseResponse {
	return BaseResponse{Code: err.Code, Message: err.Message}
}

func ErrorWithMsg(err *AppError, message string) BaseResponse {
	return Error(err.WithMessage(message))
}

func WriteSuccess(c *gin.Context, httpStatus int, data any) {
	c.JSON(httpStatus, Success(data))
}

func WriteSuccessWithMsg(c *gin.Context, httpStatus int, data any, message string) {
	c.JSON(httpStatus, SuccessWithMsg(data, message))
}

func WriteError(c *gin.Context, err *AppError) {
	c.JSON(err.HTTPCode, Error(err))
}

// WriteBindError maps an oversized body to 413 and other binding errors to 400.
func WriteBindError(c *gin.Context, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		WriteError(c, ErrRequestTooLarge)
		return
	}
	WriteError(c, ErrParams)
}

// WriteErrorFrom preserves known business errors and hides unexpected details.
func WriteErrorFrom(c *gin.Context, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		WriteError(c, appErr)
		return
	}
	log.Printf("request failed: %v", err)
	WriteError(c, ErrSystem)
}
