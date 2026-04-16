package alert

import (
	"fmt"
	"os/exec"
	"runtime"
	"stock-monitor/internal/log"
	"stock-monitor/internal/types"
	"strings"
)

// NotificationParams contains parameters for alert notifications
type NotificationParams struct {
	Alert   types.Alert
	GetText func(key string) string
}

// SendNotification sends cross-platform notification for an alert
func SendNotification(params NotificationParams) {
	var title, message string

	switch params.Alert.Type {
	case types.AlertTypePrice:
		title = params.GetText("alertNotificationPriceTitle")
		message = fmt.Sprintf("%s (%s) %s %s %.2f",
			params.Alert.StockName, params.Alert.StockCode,
			params.GetText("alertNotificationPrice"),
			params.Alert.Condition, params.Alert.Threshold)
	case types.AlertTypeRate:
		title = params.GetText("alertNotificationRateTitle")
		message = fmt.Sprintf("%s (%s) %s %s %.2f%%",
			params.Alert.StockName, params.Alert.StockCode,
			params.GetText("alertNotificationRate"),
			params.Alert.Condition, params.Alert.Threshold)
	case types.AlertTypeVolume:
		title = params.GetText("alertNotificationVolumeTitle")
		message = fmt.Sprintf("%s (%s) %s %s %.0f",
			params.Alert.StockName, params.Alert.StockCode,
			params.GetText("alertNotificationVolume"),
			params.Alert.Condition, params.Alert.Threshold)
	}

	// Platform-specific notification
	switch runtime.GOOS {
	case "darwin": // macOS
		sendMacOSNotification(title, message)
	case "linux": // Linux
		sendLinuxNotification(title, message)
	case "windows": // Windows
		sendWindowsNotification(title, message)
	default:
		log.Warn("log.alert.unsupportedPlatform", runtime.GOOS)
	}
}

// sendMacOSNotification sends macOS notification via terminal-notifier
func sendMacOSNotification(title, message string) {
	binaryPath, err := exec.LookPath("terminal-notifier")
	if err != nil {
		log.Warn("log.alert.terminalNotifierNotFound", "")
		return
	}

	cmd := exec.Command(
		binaryPath,
		"-title", title,
		"-message", message,
		"-sound", "default",
	)

	if err := cmd.Run(); err != nil {
		log.Error("log.alert.notificationFailed", err)
	} else {
		log.Info("log.alert.notificationSent", title)
	}
}

// sendLinuxNotification sends Linux notification via notify-send
func sendLinuxNotification(title, message string) {
	cmd := exec.Command("notify-send", title, message)
	if err := cmd.Run(); err != nil {
		log.Warn("log.alert.notifySendNotFound", "")
	} else {
		log.Info("log.alert.notificationSent", title)
	}
}

// sendWindowsNotification sends Windows notification via PowerShell.
// Uses single-quoted strings to prevent variable expansion and subexpression injection (CWE-78).
func sendWindowsNotification(title, message string) {
	escapedMessage := strings.ReplaceAll(message, "'", "''")
	escapedTitle := strings.ReplaceAll(title, "'", "''")
	psScript := fmt.Sprintf(
		`[System.Reflection.Assembly]::LoadWithPartialName('System.Windows.Forms'); [System.Windows.Forms.MessageBox]::Show('%s', '%s')`,
		escapedMessage, escapedTitle,
	)
	cmd := exec.Command("powershell", "-Command", psScript)
	if err := cmd.Run(); err != nil {
		log.Warn("log.alert.powershellNotFound", "")
	} else {
		log.Info("log.alert.notificationSent", title)
	}
}
