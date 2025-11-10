// Package backend 提供编译后端支持
// 负责将 LLVM IR 编译为可执行文件
package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// CompileOptions 编译选项（单文件编译）
type CompileOptions struct {
	InputFile  string // 输入的 LLVM IR 文件路径
	OutputFile string // 输出的可执行文件路径
	TargetOS   string // 目标操作系统 (windows/linux/darwin)
	TargetArch string // 目标架构 (x86_64/aarch64)
	Optimize   bool   // 是否开启优化
}

// LinkOptions 链接选项（多文件链接）
type LinkOptions struct {
	InputFiles []string // 输入的 LLVM IR 文件路径列表
	OutputFile string   // 输出的可执行文件路径
	TargetOS   string   // 目标操作系统 (windows/linux/darwin)
	TargetArch string   // 目标架构 (x86_64/aarch64)
	Optimize   bool     // 是否开启优化
}

// ZigCompiler Zig 编译器封装
type ZigCompiler struct {
	zigPath string // zig 可执行文件路径
}

// NewZigCompiler 创建新的 Zig 编译器实例
// 自动查找 zig 可执行文件，按以下顺序：
// 1. 环境变量 GOC_ZIG_PATH
// 2. 项目目录下的 backend/binaries/zig-*/zig.exe
// 3. 系统 PATH
func NewZigCompiler() (*ZigCompiler, error) {
	zigPath, err := findZigExecutable()
	if err != nil {
		return nil, err
	}

	return &ZigCompiler{
		zigPath: zigPath,
	}, nil
}

// Compile 编译 LLVM IR 文件为可执行文件
// 使用 zig cc 命令，它是 clang 的包装器
func (z *ZigCompiler) Compile(opts CompileOptions) error {
	// 设置默认值
	if opts.TargetOS == "" {
		opts.TargetOS = runtime.GOOS
	}
	if opts.TargetArch == "" {
		opts.TargetArch = runtime.GOARCH
	}

	// 构建 zig cc 命令
	// zig cc 是一个功能强大的 clang 包装器，支持交叉编译
	args := []string{
		"cc",           // 使用 zig 的 C 编译器模式
		opts.InputFile, // 输入 LLVM IR 文件
		"-o", opts.OutputFile, // 输出文件
	}

	// 添加目标平台参数
	target := z.buildTargetTriple(opts.TargetOS, opts.TargetArch)
	if target != "" {
		args = append(args, "-target", target)
	}

	// 添加优化选项
	if opts.Optimize {
		args = append(args, "-O2")
	}

	// 执行 zig cc 命令
	cmd := exec.Command(z.zigPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("zig compilation failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

// Link 链接多个 LLVM IR 文件为可执行文件
// 使用 zig cc 命令链接多个文件
func (z *ZigCompiler) Link(opts LinkOptions) error {
	// 设置默认值
	if opts.TargetOS == "" {
		opts.TargetOS = runtime.GOOS
	}
	if opts.TargetArch == "" {
		opts.TargetArch = runtime.GOARCH
	}

	// 构建 zig cc 命令
	args := []string{"cc"}

	// 添加所有输入文件
	args = append(args, opts.InputFiles...)

	// 输出文件
	args = append(args, "-o", opts.OutputFile)

	// 添加目标平台参数
	target := z.buildTargetTriple(opts.TargetOS, opts.TargetArch)
	if target != "" {
		args = append(args, "-target", target)
	}

	// 添加优化选项
	if opts.Optimize {
		args = append(args, "-O2")
	}

	// 执行 zig cc 命令
	cmd := exec.Command(z.zigPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("zig linking failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

// buildTargetTriple 构建目标平台三元组
// 格式: <arch>-<os>-<abi>
// 例如: x86_64-windows-gnu, x86_64-linux-musl, aarch64-macos-none
func (z *ZigCompiler) buildTargetTriple(targetOS, targetArch string) string {
	arch := targetArch
	if targetArch == "amd64" {
		arch = "x86_64"
	} else if targetArch == "arm64" {
		arch = "aarch64"
	}

	switch targetOS {
	case "windows":
		return fmt.Sprintf("%s-windows-gnu", arch)
	case "linux":
		return fmt.Sprintf("%s-linux-musl", arch)
	case "darwin":
		return fmt.Sprintf("%s-macos-none", arch)
	default:
		return ""
	}
}

// GetVersion 获取 zig 版本信息
func (z *ZigCompiler) GetVersion() (string, error) {
	cmd := exec.Command(z.zigPath, "version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get zig version: %v", err)
	}
	return string(output), nil
}

// findZigExecutable 查找 zig 可执行文件
// 按以下顺序查找：
// 1. 环境变量 GOC_ZIG_PATH
// 2. 项目目录下的 backend/binaries/zig-*/zig 或 zig.exe
// 3. 系统 PATH 中的 zig
func findZigExecutable() (string, error) {
	// 1. 检查环境变量
	if zigPath := os.Getenv("GOC_ZIG_PATH"); zigPath != "" {
		if _, err := os.Stat(zigPath); err == nil {
			return zigPath, nil
		}
	}

	// 2. 检查项目目录下的 backend/binaries/zig-*/zig.exe
	executableName := "zig"
	if runtime.GOOS == "windows" {
		executableName = "zig.exe"
	}

	// 获取项目根目录（假设 backend 包在项目的 backend 目录下）
	// 我们需要向上查找到项目根目录
	binariesDir := "backend/binaries"

	// 尝试在 binaries 目录下查找 zig-* 子目录
	matches, err := filepath.Glob(filepath.Join(binariesDir, "zig-*", executableName))
	if err == nil && len(matches) > 0 {
		// 返回第一个匹配的路径
		absPath, err := filepath.Abs(matches[0])
		if err == nil {
			return absPath, nil
		}
	}

	// 3. 在系统 PATH 中查找
	zigPath, err := exec.LookPath("zig")
	if err == nil {
		return zigPath, nil
	}

	// 没有找到 zig
	return "", fmt.Errorf("zig compiler not found. Please:\n" +
		"  1. Download zig from https://ziglang.org/download/\n" +
		"  2. Extract to backend/binaries/zig-<platform>/ directory, OR\n" +
		"  3. Add zig to your system PATH, OR\n" +
		"  4. Set GOC_ZIG_PATH environment variable to zig executable path")
}
