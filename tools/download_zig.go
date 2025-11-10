// tools/download_zig.go - 下载 zig 编译器工具
//
// 使用方法:
//   go run tools/download_zig.go
//
// 此脚本会下载所有支持平台的 zig 编译器二进制文件
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const (
	// Zig 版本
	zigVersion = "0.13.0"

	// 下载基础 URL
	baseURL = "https://ziglang.org/download/%s/zig-%s-%s.%s"
)

// Platform 平台信息
type Platform struct {
	OS       string // 操作系统
	Arch     string // 架构
	ZigOS    string // Zig 平台名称
	ZigArch  string // Zig 架构名称
	Ext      string // 压缩包扩展名
	BinName  string // 输出的二进制文件名
}

var platforms = []Platform{
	{
		OS:      "windows",
		Arch:    "amd64",
		ZigOS:   "windows",
		ZigArch: "x86_64",
		Ext:     "zip",
		BinName: "zig-windows-x86_64.exe",
	},
	{
		OS:      "linux",
		Arch:    "amd64",
		ZigOS:   "linux",
		ZigArch: "x86_64",
		Ext:     "tar.xz",
		BinName: "zig-linux-x86_64",
	},
	{
		OS:      "darwin",
		Arch:    "amd64",
		ZigOS:   "macos",
		ZigArch: "x86_64",
		Ext:     "tar.xz",
		BinName: "zig-macos-x86_64",
	},
	{
		OS:      "darwin",
		Arch:    "arm64",
		ZigOS:   "macos",
		ZigArch: "aarch64",
		Ext:     "tar.xz",
		BinName: "zig-macos-aarch64",
	},
}

func main() {
	fmt.Println("GoC Zig Compiler Downloader")
	fmt.Println("============================")
	fmt.Printf("Downloading Zig %s for all supported platforms...\n\n", zigVersion)

	// 创建输出目录
	outputDir := "backend/binaries"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// 下载每个平台的 zig
	for _, platform := range platforms {
		fmt.Printf("Downloading Zig for %s/%s...\n", platform.OS, platform.Arch)

		if err := downloadAndExtractZig(platform, outputDir); err != nil {
			fmt.Printf("  ❌ Error: %v\n", err)
			continue
		}

		fmt.Printf("  ✅ Success: %s\n\n", platform.BinName)
	}

	fmt.Println("Download complete!")
	fmt.Printf("\nAll binaries saved to: %s/\n", outputDir)
}

// downloadAndExtractZig 下载并提取 zig 二进制文件
func downloadAndExtractZig(platform Platform, outputDir string) error {
	// 构建下载 URL
	archiveName := fmt.Sprintf("zig-%s-%s", platform.ZigOS, platform.ZigArch)
	url := fmt.Sprintf(baseURL, zigVersion, platform.ZigOS, platform.ZigArch, platform.Ext)

	// 创建临时文件
	tempFile := filepath.Join(os.TempDir(), archiveName+"."+platform.Ext)
	defer os.Remove(tempFile)

	// 下载文件
	if err := downloadFile(url, tempFile); err != nil {
		return fmt.Errorf("download failed: %v", err)
	}

	// 提取 zig 可执行文件
	executableName := "zig"
	if platform.OS == "windows" {
		executableName = "zig.exe"
	}

	extractedPath := ""
	if platform.Ext == "zip" {
		extractedPath, _ = extractFromZip(tempFile, executableName, archiveName)
	} else if platform.Ext == "tar.xz" {
		// 对于 tar.xz，我们先跳过 xz 解压（需要额外的库）
		// 建议用户手动下载或使用其他方法
		return fmt.Errorf("tar.xz format not yet supported, please download manually from: %s", url)
	}

	if extractedPath == "" {
		return fmt.Errorf("failed to extract zig executable")
	}

	// 移动到目标位置
	targetPath := filepath.Join(outputDir, platform.BinName)
	if err := os.Rename(extractedPath, targetPath); err != nil {
		// 如果重命名失败，尝试复制
		if err := copyFile(extractedPath, targetPath); err != nil {
			return fmt.Errorf("failed to move binary: %v", err)
		}
		os.Remove(extractedPath)
	}

	// 设置可执行权限（Unix-like 系统）
	if runtime.GOOS != "windows" {
		if err := os.Chmod(targetPath, 0755); err != nil {
			return fmt.Errorf("failed to set executable permission: %v", err)
		}
	}

	return nil
}

// downloadFile 下载文件
func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// 显示下载进度
	_, err = io.Copy(out, resp.Body)
	return err
}

// extractFromZip 从 zip 文件中提取特定文件
func extractFromZip(zipPath, fileName, archiveName string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		// zig 压缩包内部结构: zig-<os>-<arch>/zig.exe
		expectedPath := archiveName + "/" + fileName
		if f.Name == expectedPath {
			// 提取到临时目录
			tempPath := filepath.Join(os.TempDir(), fileName)

			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			out, err := os.Create(tempPath)
			if err != nil {
				return "", err
			}
			defer out.Close()

			_, err = io.Copy(out, rc)
			if err != nil {
				return "", err
			}

			return tempPath, nil
		}
	}

	return "", fmt.Errorf("file not found in archive: %s", fileName)
}

// extractFromTarGz 从 tar.gz 文件中提取特定文件
func extractFromTarGz(tarGzPath, fileName, archiveName string) (string, error) {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		expectedPath := archiveName + "/" + fileName
		if header.Name == expectedPath {
			tempPath := filepath.Join(os.TempDir(), fileName)

			out, err := os.Create(tempPath)
			if err != nil {
				return "", err
			}
			defer out.Close()

			_, err = io.Copy(out, tr)
			if err != nil {
				return "", err
			}

			return tempPath, nil
		}
	}

	return "", fmt.Errorf("file not found in archive: %s", fileName)
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
