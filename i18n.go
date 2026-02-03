package main

import (
	"stock-monitor/internal/consts"
	"embed"
	"encoding/json"
	"fmt"
	"os"
)

//go:embed i18n/*.json
var embeddedI18n embed.FS

// texts i18n 配置 - 存储各语言的文本映射
var texts map[Language]TextMap

// loadLanguageFile 加载单个语言文件（优先外部文件，失败时使用嵌入文件）
func loadLanguageFile(filename string, lang Language) bool {
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

// loadI18nFiles 加载 i18n 文件
func loadI18nFiles() {
	texts = make(map[Language]TextMap)

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

// getText 获取本地化文本的辅助函数
func (m *Model) getText(key string) string {
	if text, exists := texts[m.language][key]; exists {
		return text
	}
	// 如果找不到文本，返回英文版本作为备用
	if text, exists := texts[consts.English][key]; exists {
		return text
	}
	return key // 最后备用返回key本身
}

// getTextForLang 获取指定语言的本地化文本（用于初始化时无Model实例的场景）
func getTextForLang(key string, lang Language) string {
	if text, exists := texts[lang][key]; exists {
		return text
	}
	// 如果找不到文本，返回英文版本作为备用
	if text, exists := texts[consts.English][key]; exists {
		return text
	}
	return key // 最后备用返回key本身
}
