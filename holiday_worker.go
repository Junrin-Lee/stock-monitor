package main

import (
	"stock-monitor/internal/market"
)

// ============================================================================
// 节假日数据同步 - 封装 internal/market 包
// ============================================================================

// mainLogger 适配器 - 将 main 包的日志函数适配到 market.Logger 接口
type mainLogger struct{}

func (l *mainLogger) Info(key string, args ...interface{}) {
	logInfo(key, args...)
}

func (l *mainLogger) Warn(key string, args ...interface{}) {
	logWarn(key, args...)
}

func (l *mainLogger) Error(key string, args ...interface{}) {
	logError(key, args...)
}

func (l *mainLogger) Debug(format string, args ...interface{}) {
	// main 包的 logDebug 需要 i18n key，这里直接传递
	if len(args) > 0 {
		logDebug(format, args...)
	} else {
		logDebug(format)
	}
}

func (l *mainLogger) WarnDirect(format string, args ...interface{}) {
	logWarn(format, args...)
}

func (l *mainLogger) ErrorDirect(format string, args ...interface{}) {
	logError(format, args...)
}

func (l *mainLogger) DebugDirect(format string, args ...interface{}) {
	logDebug(format, args...)
}

// StartHolidayWorker 启动节假日数据同步工作器
func StartHolidayWorker() {
	// 创建 market 包的 HolidayWorker，传入日志适配器
	worker := market.NewHolidayWorker(&mainLogger{})

	// 启动工作器
	worker.Start()
}
