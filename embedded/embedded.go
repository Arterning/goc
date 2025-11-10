// Package embedded 提供内嵌资源管理
// 将 NASM 等外部工具打包到编译器中，实现完全自包含
package embedded

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

//go:embed tools/*
var toolsFS embed.FS

var (
	// 缓存已提取的工具路径
	extractedTools = make(map[string]string)
	mu             sync.Mutex
)

// GetNASMPath 获取 NASM 可执行文件的路径
// 如果是嵌入版本，会提取到临时目录
// 如果嵌入文件不存在，会尝试使用系统路径
func GetNASMPath() (string, error) {
	mu.Lock()
	defer mu.Unlock()

	// 检查缓存
	if path, ok := extractedTools["nasm"]; ok {
		return path, nil
	}

	// 确定文件名
	var embeddedName string
	if runtime.GOOS == "windows" {
		embeddedName = "tools/nasm.exe"
	} else {
		embeddedName = "tools/nasm"
	}

	// 尝试读取嵌入的文件
	data, err := toolsFS.ReadFile(embeddedName)
	if err != nil {
		// 嵌入文件不存在，尝试使用系统路径
		return "", fmt.Errorf("embedded NASM not found (please download and place in embedded/tools/), error: %w", err)
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "goc-tools-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// 提取到临时文件
	var exeName string
	if runtime.GOOS == "windows" {
		exeName = "nasm.exe"
	} else {
		exeName = "nasm"
	}

	exePath := filepath.Join(tempDir, exeName)
	if err := os.WriteFile(exePath, data, 0755); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to extract NASM: %w", err)
	}

	// 缓存路径
	extractedTools["nasm"] = exePath

	return exePath, nil
}

// Cleanup 清理临时提取的工具文件
// 应在程序退出时调用
func Cleanup() error {
	mu.Lock()
	defer mu.Unlock()

	for _, path := range extractedTools {
		dir := filepath.Dir(path)
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}

	extractedTools = make(map[string]string)
	return nil
}

// IsEmbedded 检查 NASM 是否已嵌入
func IsEmbedded() bool {
	var embeddedName string
	if runtime.GOOS == "windows" {
		embeddedName = "tools/nasm.exe"
	} else {
		embeddedName = "tools/nasm"
	}

	_, err := toolsFS.ReadFile(embeddedName)
	return err == nil
}
