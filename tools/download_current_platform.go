// tools/download_current_platform.go - 下载当前平台的 Zig 编译器
//
// 使用方法:
//   go run tools/download_current_platform.go
//
// 此脚本只下载当前运行平台的 zig 编译器，更快更简单
package main

import (
	"archive/zip"
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
)

func main() {
	fmt.Println("GoC Zig Compiler Downloader (Current Platform)")
	fmt.Println("===============================================")
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Zig Version: %s\n\n", zigVersion)

	// 确定当前平台
	platform, err := getCurrentPlatform()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// 创建输出目录
	outputDir := "backend/binaries"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// 检查是否已经存在
	outputPath := filepath.Join(outputDir, platform.BinName)
	if _, err := os.Stat(outputPath); err == nil {
		fmt.Printf("Zig binary already exists at: %s\n", outputPath)
		fmt.Println("Delete it first if you want to re-download.")
		return
	}

	fmt.Printf("Downloading Zig for %s/%s...\n", platform.OS, platform.Arch)

	if err := downloadAndExtractZig(platform, outputDir); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Success! Zig binary saved to: %s\n", outputPath)
	fmt.Println("\nYou can now build and run GoC:")
	fmt.Println("  go build")
	fmt.Println("  ./goc examples/hello.goc")
}

type Platform struct {
	OS       string
	Arch     string
	ZigOS    string
	ZigArch  string
	Ext      string
	BinName  string
}

func getCurrentPlatform() (Platform, error) {
	platforms := map[string]map[string]Platform{
		"windows": {
			"amd64": {
				OS:      "windows",
				Arch:    "amd64",
				ZigOS:   "windows",
				ZigArch: "x86_64",
				Ext:     "zip",
				BinName: "zig-windows-x86_64.exe",
			},
		},
		"linux": {
			"amd64": {
				OS:      "linux",
				Arch:    "amd64",
				ZigOS:   "linux",
				ZigArch: "x86_64",
				Ext:     "tar.xz",
				BinName: "zig-linux-x86_64",
			},
		},
		"darwin": {
			"amd64": {
				OS:      "darwin",
				Arch:    "amd64",
				ZigOS:   "macos",
				ZigArch: "x86_64",
				Ext:     "tar.xz",
				BinName: "zig-macos-x86_64",
			},
			"arm64": {
				OS:      "darwin",
				Arch:    "arm64",
				ZigOS:   "macos",
				ZigArch: "aarch64",
				Ext:     "tar.xz",
				BinName: "zig-macos-aarch64",
			},
		},
	}

	osPlatforms, ok := platforms[runtime.GOOS]
	if !ok {
		return Platform{}, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	platform, ok := osPlatforms[runtime.GOARCH]
	if !ok {
		return Platform{}, fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}

	return platform, nil
}

func downloadAndExtractZig(platform Platform, outputDir string) error {
	// 构建下载 URL
	archiveName := fmt.Sprintf("zig-%s-%s", platform.ZigOS, platform.ZigArch)
	url := fmt.Sprintf("https://ziglang.org/download/%s/%s-%s.%s",
		zigVersion, archiveName, zigVersion, platform.Ext)

	fmt.Printf("Download URL: %s\n", url)

	// 仅支持 zip 格式（Windows）
	if platform.Ext != "zip" {
		return fmt.Errorf("automatic download only supports Windows (zip format)\n" +
			"For Linux/macOS, please download manually from: " + url + "\n" +
			"See SETUP.md for instructions")
	}

	// 下载到临时文件
	tempFile := filepath.Join(os.TempDir(), archiveName+".zip")
	defer os.Remove(tempFile)

	fmt.Println("Downloading... (this may take a few minutes, ~50MB)")
	if err := downloadFile(url, tempFile); err != nil {
		return fmt.Errorf("download failed: %v", err)
	}

	fmt.Println("Download complete. Extracting...")

	// 提取 zig.exe
	executableName := "zig.exe"
	extractedPath, err := extractFromZip(tempFile, executableName, archiveName)
	if err != nil {
		return fmt.Errorf("extraction failed: %v", err)
	}
	defer os.Remove(extractedPath)

	// 移动到目标位置
	targetPath := filepath.Join(outputDir, platform.BinName)
	if err := os.Rename(extractedPath, targetPath); err != nil {
		if err := copyFile(extractedPath, targetPath); err != nil {
			return fmt.Errorf("failed to move binary: %v", err)
		}
	}

	return nil
}

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

	// 显示进度
	size := resp.ContentLength
	downloaded := int64(0)
	buf := make([]byte, 32*1024)

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
			if size > 0 {
				percent := float64(downloaded) / float64(size) * 100
				fmt.Printf("\rProgress: %.1f%% (%d/%d MB)", percent,
					downloaded/1024/1024, size/1024/1024)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	fmt.Println()

	return nil
}

func extractFromZip(zipPath, fileName, archiveName string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// zig 压缩包内部结构: zig-<os>-<arch>-<version>/zig.exe
	expectedPath := fmt.Sprintf("%s-%s/%s", archiveName, zigVersion, fileName)

	for _, f := range r.File {
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

	return "", fmt.Errorf("file not found in archive: %s", expectedPath)
}

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
