package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"

	"stock-monitor/internal/consts"
)

// embeddedI18n 嵌入的 i18n 文件系统（由主包设置）
var embeddedI18n embed.FS

// SetEmbeddedFS 设置嵌入的文件系统
func SetEmbeddedFS(fs embed.FS) {
	embeddedI18n = fs
}

// TextMap 文本映射结构
type TextMap map[string]string

// texts i18n 配置 - 存储各语言的文本映射
var texts map[consts.Language]TextMap

// currentLanguage 当前语言
var currentLanguage consts.Language = consts.Chinese

// loadLanguageFile 加载单个语言文件（优先外部文件，失败时使用嵌入文件）
func loadLanguageFile(filename string, lang consts.Language) bool {
	var data []byte
	var err error

	// 1. 尝试从外部文件读取（开发环境）
	externalPath := "i18n/" + filename
	data, err = os.ReadFile(externalPath)
	if err != nil {
		// 2. 外部文件不存在，尝试从嵌入文件读取（生产环境）
		embeddedPath := "i18n/" + filename
		data, err = embeddedI18n.ReadFile(embeddedPath)
		if err != nil {
			fmt.Printf("Warning: Failed to read %s from both external and embedded sources: %v\n", filename, err)
			return false
		}
	}

	// 解析 JSON
	var langTexts TextMap
	if err := json.Unmarshal(data, &langTexts); err != nil {
		fmt.Printf("Warning: Failed to parse %s: %v\n", filename, err)
		return false
	}

	texts[lang] = langTexts
	return true
}

// LoadI18nFiles 加载 i18n 文件
func LoadI18nFiles() {
	texts = make(map[consts.Language]TextMap)

	// 加载中文配置
	loadLanguageFile("zh.json", consts.Chinese)

	// 加载英文配置
	loadLanguageFile("en.json", consts.English)

	// 如果没有成功加载任何语言文件，退出程序
	if len(texts) == 0 {
		fmt.Println("Error: No i18n files could be loaded. Please ensure i18n files are available.")
		os.Exit(1)
	}
}

// SetLanguage 设置当前语言
func SetLanguage(lang consts.Language) {
	currentLanguage = lang
}

// GetLanguage 获取当前语言
func GetLanguage() consts.Language {
	return currentLanguage
}

// GetText 获取本地化文本
func GetText(key string) string {
	return GetTextWithLang(key, currentLanguage)
}

// GetTextWithLang 获取指定语言的本地化文本
func GetTextWithLang(key string, lang consts.Language) string {
	if texts == nil {
		return key
	}
	if text, exists := texts[lang][key]; exists {
		return text
	}
	// 如果找不到文本，返回英文版本作为备用
	if text, exists := texts[consts.English][key]; exists {
		return text
	}
	return key // 最后备用返回key本身
}

// GetTexts 获取所有文本映射（用于 main 包的兼容性）
func GetTexts() map[consts.Language]TextMap {
	return texts
}
