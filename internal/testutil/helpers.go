package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// CreateTempDir 创建临时测试目录
func CreateTempDir(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "stock-monitor-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})
	return tmpDir
}

// CreateTempFile 创建临时测试文件
func CreateTempFile(t *testing.T, dir, pattern string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp(dir, pattern)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	tmpFile.Close()
	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})
	return tmpFile.Name()
}

// WriteTestFile 写入测试文件内容
func WriteTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
}

// ReadTestFile 读取测试文件内容
func ReadTestFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取测试文件失败: %v", err)
	}
	return string(content)
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// MustMkdirAll 创建目录(测试专用)
func MustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
}

// GetTestDataPath 获取测试数据路径
func GetTestDataPath(t *testing.T, filename string) string {
	t.Helper()
	tmpDir := CreateTempDir(t)
	return filepath.Join(tmpDir, filename)
}

// CompareFloats 比较浮点数(带容差)
func CompareFloats(a, b, epsilon float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}

// FloatEquals 检查两个浮点数是否相等(默认容差 0.0001)
func FloatEquals(a, b float64) bool {
	return CompareFloats(a, b, 0.0001)
}

// StringSliceEqual 检查两个字符串切片是否相等
func StringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ContainsString 检查切片是否包含字符串
func ContainsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
