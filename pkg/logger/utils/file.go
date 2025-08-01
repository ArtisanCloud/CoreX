package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// 确保文件路径存在的函数
func EnsureFileExists(filePath string) {
	if filePath == "" {
		return
	}

	// 获取日志文件所在的目录
	dir := filepath.Dir(filePath)

	// 创建目录（如果不存在）
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		fmt.Printf("无法创建日志目录: %v\n", err)
	}

	// 创建日志文件（如果不存在）
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Printf("无法创建日志文件: %v\n", err)
		}
		file.Close()
	}
}
