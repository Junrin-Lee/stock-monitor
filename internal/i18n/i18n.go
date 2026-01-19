package i18n

import (
	"encoding/json"
	"fmt"
	"os"

	"stock-monitor/internal/consts"
)

// TextMap 文本映射结构
type TextMap map[string]string

// texts i18n 配置 - 存储各语言的文本映射
var texts map[consts.Language]TextMap

// currentLanguage 当前语言
var currentLanguage consts.Language = consts.Chinese

// LoadI18nFiles 加载 i18n 文件
func LoadI18nFiles() {
	texts = make(map[consts.Language]TextMap)

	// 读取中文配置
	if zhData, err := os.ReadFile("i18n/zh.json"); err == nil {
		var zhTexts TextMap
		if err := json.Unmarshal(zhData, &zhTexts); err == nil {
			texts[consts.Chinese] = zhTexts
		} else {
			fmt.Printf("Warning: Failed to parse i18n/zh.json: %v\n", err)
		}
	} else {
		fmt.Printf("Warning: Failed to read i18n/zh.json: %v\n", err)
	}

	// 读取英文配置
	if enData, err := os.ReadFile("i18n/en.json"); err == nil {
		var enTexts TextMap
		if err := json.Unmarshal(enData, &enTexts); err == nil {
			texts[consts.English] = enTexts
		} else {
			fmt.Printf("Warning: Failed to parse i18n/en.json: %v\n", err)
		}
	} else {
		fmt.Printf("Warning: Failed to read i18n/en.json: %v\n", err)
	}

	// 如果没有成功加载任何语言文件，退出程序
	if len(texts) == 0 {
		fmt.Println("Error: No i18n files could be loaded. Please ensure i18n/zh.json and i18n/en.json exist.")
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
