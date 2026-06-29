package errors

import (
	"fmt"
	"stock-monitor/internal/log"
)

// ErrorType 错误类型
type ErrorType string

const (
	ErrTypeAPI        ErrorType = "api"
	ErrTypeData       ErrorType = "data"
	ErrTypeValidation ErrorType = "validation"
	ErrTypeNotFound   ErrorType = "not_found"
	ErrTypePermission ErrorType = "permission"
	ErrTypeTimeout    ErrorType = "timeout"
)

// AppError 应用错误
type AppError struct {
	Type    ErrorType
	Message string
	Err     error
	Context map[string]interface{}
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap 支持 errors.Unwrap
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError 创建应用错误
func NewAppError(errType ErrorType, message string, err error) *AppError {
	return &AppError{
		Type:    errType,
		Message: message,
		Err:     err,
		Context: make(map[string]interface{}),
	}
}

// WithContext 添加上下文信息
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	e.Context[key] = value
	return e
}

// LogError 记录错误日志
func (e *AppError) LogError() {
	log.Error("app.error", e.Error())
	for k, v := range e.Context {
		log.Debug("app.error.context", k, v)
	}
}

// IsType 检查错误类型
func IsType(err error, errType ErrorType) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == errType
	}
	return false
}

// API 错误构造函数
func NewAPIError(message string, err error) *AppError {
	return NewAppError(ErrTypeAPI, message, err)
}

// 数据错误构造函数
func NewDataError(message string, err error) *AppError {
	return NewAppError(ErrTypeData, message, err)
}

// 验证错误构造函数
func NewValidationError(message string) *AppError {
	return NewAppError(ErrTypeValidation, message, nil)
}

// 未找到错误构造函数
func NewNotFoundError(resource string) *AppError {
	return NewAppError(ErrTypeNotFound, fmt.Sprintf("%s not found", resource), nil)
}

// 超时错误构造函数
func NewTimeoutError(operation string) *AppError {
	return NewAppError(ErrTypeTimeout, fmt.Sprintf("%s timeout", operation), nil)
}
