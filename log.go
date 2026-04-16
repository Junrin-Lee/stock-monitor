package main

import (
	"stock-monitor/internal/log"
)

// ============================================================================
// 日志系统 - 封装 internal/log 包
// ============================================================================

// LogLevel 日志级别类型
type LogLevel = int

// 日志级别常量
const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
)

// Logger 日志封装器（向后兼容）
type Logger struct {
	// 空结构体，只是为了向后兼容
}

// globalLogger 全局日志实例（为了向后兼容 main.go 中的 defer globalLogger.Sync()）
var globalLogger *Logger

// ============================================================================
// 初始化
// ============================================================================

// InitLogger 初始化全局日志系统
func InitLogger(logDir string, level LogLevel) error {
	// 调用 internal/log 包的初始化
	var internalLevel log.LogLevel
	switch level {
	case LogDebug:
		internalLevel = log.LogDebug
	case LogInfo:
		internalLevel = log.LogInfo
	case LogWarn:
		internalLevel = log.LogWarn
	case LogError:
		internalLevel = log.LogError
	default:
		internalLevel = log.LogInfo
	}

	if err := log.InitLogger(logDir, internalLevel); err != nil {
		return err
	}

	// 设置 i18n 文本获取器（从 globalModel 获取，使用 atomic 读取保证并发安全）
	log.SetTextGetter(func(key string) string {
		if m := globalModel.Load(); m != nil {
			return m.getText(key)
		}
		return key
	})

	// 为了向后兼容，创建一个 globalLogger 实例
	globalLogger = &Logger{}

	return nil
}

// ============================================================================
// Logger 方法（向后兼容）
// ============================================================================

// Sync 刷新日志缓冲区
func (l *Logger) Sync() {
	log.GlobalSync()
}

// ============================================================================
// 日志函数 - 四个级别（封装 internal/log 包）
// ============================================================================

// logDebug DEBUG 级别日志 - 详细调试信息
func logDebug(key string, args ...any) {
	log.Debug(key, args...)
}

// logInfo INFO 级别日志 - 正常运行信息
func logInfo(key string, args ...any) {
	log.Info(key, args...)
}

// logWarn WARN 级别日志 - 可能的问题
func logWarn(key string, args ...any) {
	log.Warn(key, args...)
}

// logError ERROR 级别日志 - 需要关注的错误
func logError(key string, args ...any) {
	log.Error(key, args...)
}

// ============================================================================
// 直接日志函数（无 i18n key）
// ============================================================================

// logInfoDirect 直接记录 INFO 级别消息（无 key）
func logInfoDirect(format string, args ...any) {
	log.InfoDirect(format, args...)
}

// logDebugDirect 直接记录 DEBUG 级别消息（无 key）
func logDebugDirect(format string, args ...any) {
	log.DebugDirect(format, args...)
}

// logWarnDirect 直接记录 WARN 级别消息（无 key）
func logWarnDirect(format string, args ...any) {
	log.WarnDirect(format, args...)
}

// logErrorDirect 直接记录 ERROR 级别消息（无 key）
func logErrorDirect(format string, args ...any) {
	log.ErrorDirect(format, args...)
}

// ============================================================================
// 辅助函数
// ============================================================================

