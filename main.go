// Package main 是 GoC 编译器的入口点
// GoC 是一个类 C 语言编译器，使用 Go 语言编写
//
// 编译流程：
// 源代码 -> 词法分析 -> 语法分析 -> 语义分析 -> 代码生成 -> 可执行文件
//
// 支持两种后端：
// 1. LLVM 后端：生成 LLVM IR，使用 Clang 编译
// 2. x64 后端：直接生成 x86-64 机器码（无外部依赖）
//
// 使用方法：
//   goc <source-file> [options]
//   例如：goc examples/hello.goc
//         goc examples/hello.goc -backend=x64
//
// 输出：
//   LLVM 后端:
//     - <source-file>.ll    LLVM IR 中间代码
//     - <source-file>.exe   可执行文件
//   x64 后端:
//     - <source-file>.asm   NASM 汇编代码
//     - <source-file>.exe   可执行文件
package main

import (
	"flag"
	"fmt"
	"goc/codegen"
	"goc/embedded"
	"goc/lexer"
	"goc/parser"
	"goc/semantic"
	"goc/x64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// main 编译器主函数
// 负责：
// 1. 解析命令行参数
// 2. 读取源文件
// 3. 调用编译流程
// 4. 处理错误
func main() {
	// 清理临时提取的工具文件
	defer embedded.Cleanup()

	// 定义命令行参数
	backend := flag.String("backend", "llvm", "Code generation backend: llvm, x64, or native")
	flag.Parse()

	// 检查是否提供了源文件
	if flag.NArg() < 1 {
		fmt.Println("Usage: goc <source-file> [options]")
		fmt.Println("Example: goc program.goc")
		fmt.Println("         goc program.goc -backend=x64")
		fmt.Println("         goc program.goc -backend=native (no external dependencies!)")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	sourceFile := flag.Arg(0)

	// 验证后端选项
	if *backend != "llvm" && *backend != "x64" && *backend != "native" {
		fmt.Printf("Invalid backend: %s (must be 'llvm', 'x64', or 'native')\n", *backend)
		os.Exit(1)
	}

	// 读取源文件内容
	source, err := os.ReadFile(sourceFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// 执行编译
	if err := compile(string(source), sourceFile, *backend); err != nil {
		fmt.Printf("Compilation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Compilation successful!")
}

// compile 编译源代码的核心函数
// 参数 source: 源代码字符串
// 参数 sourceFile: 源文件路径（用于生成输出文件名）
// 参数 backend: 后端选择（"llvm" 或 "x64"）
// 返回值: 编译错误（如果有）
//
// 编译流程（编译器前端的经典四个阶段）：
//
// 1. 词法分析（Lexical Analysis）
//    输入：源代码字符串
//    输出：Token 流
//    功能：将字符序列分解为词法单元
//
// 2. 语法分析（Syntax Analysis）
//    输入：Token 流
//    输出：抽象语法树（AST）
//    功能：根据语法规则构建程序的树形结构
//
// 3. 语义分析（Semantic Analysis）
//    输入：AST
//    输出：类型标注的 AST
//    功能：检查类型、作用域、语义正确性
//
// 4. 代码生成（Code Generation）
//    输入：类型标注的 AST
//    输出：LLVM IR 或 x86-64 汇编
//    功能：将高级语法转换为目标代码
//
// 5. 后端编译（Backend Compilation）
//    输入：LLVM IR 或 x86-64 汇编
//    输出：可执行文件
//    工具：Clang（LLVM）或 NASM+ld（x64）
func compile(source string, sourceFile string, backend string) error {
	// ========== 阶段 1: 词法分析 ==========
	fmt.Println("=== Lexical Analysis ===")
	lex := lexer.New(source)

	// ========== 阶段 2: 语法分析 ==========
	fmt.Println("=== Syntax Analysis ===")
	p := parser.New(lex)
	program := p.ParseProgram()

	// 检查语法错误
	if len(p.Errors()) > 0 {
		fmt.Println("Parser errors:")
		for _, err := range p.Errors() {
			fmt.Printf("  %s\n", err)
		}
		return fmt.Errorf("parsing failed")
	}
	fmt.Println("Parsing successful")

	// ========== 阶段 3: 语义分析 ==========
	fmt.Println("=== Semantic Analysis ===")
	analyzer := semantic.New()
	if !analyzer.Analyze(program) {
		fmt.Println("Semantic errors:")
		for _, err := range analyzer.Errors() {
			fmt.Printf("  %s\n", err)
		}
		return fmt.Errorf("semantic analysis failed")
	}
	fmt.Println("Semantic analysis successful")

	// 获取基础文件名（不含扩展名）
	baseName := strings.TrimSuffix(sourceFile, filepath.Ext(sourceFile))

	// 根据选择的后端生成代码
	switch backend {
	case "native":
		return compileWithNativeBackend(program, analyzer, baseName)
	case "x64":
		return compileWithX64Backend(program, analyzer, baseName)
	default:
		return compileWithLLVMBackend(program, analyzer, baseName)
	}
}

// compileWithNativeBackend 使用原生机器码后端编译（完全自包含，无外部依赖）
func compileWithNativeBackend(program *parser.Program, analyzer *semantic.Analyzer, baseName string) error {
	// ========== 阶段 4: 原生机器码生成 ==========
	fmt.Println("=== Native Machine Code Generation ===")
	gen := x64.NewNativeCodeGenerator(analyzer)
	if err := gen.Generate(program); err != nil {
		return fmt.Errorf("native code generation failed: %v", err)
	}

	// ========== 阶段 5: 生成可执行文件 ==========
	fmt.Println("=== Generating Executable ===")
	exeFile := baseName + ".exe"

	if err := gen.BuildExecutable(exeFile); err != nil {
		return fmt.Errorf("failed to build executable: %v", err)
	}

	fmt.Printf("✓ Native executable created: %s\n", exeFile)
	fmt.Println("✓ No external dependencies required!")
	fmt.Printf("✓ Machine code size: %d bytes\n", len(gen.GetMachineCode()))
	return nil
}

// compileWithX64Backend 使用 x64 后端编译
func compileWithX64Backend(program *parser.Program, analyzer *semantic.Analyzer, baseName string) error {
	// ========== 阶段 4: x64 代码生成 ==========
	fmt.Println("=== x64 Code Generation ===")
	gen := x64.NewCodeGenerator(analyzer)
	if err := gen.Generate(program); err != nil {
		return fmt.Errorf("x64 code generation failed: %v", err)
	}

	// 获取生成的汇编器
	asm := gen.GetAssembler()

	// 生成 NASM 汇编代码
	builder := x64.NewExecutableBuilder(asm)
	asmCode := builder.GenerateNASM()

	// 写入汇编文件
	asmFile := baseName + ".asm"
	if err := builder.SaveAssembly(asmCode, asmFile); err != nil {
		return fmt.Errorf("failed to write assembly file: %v", err)
	}
	fmt.Printf("Assembly code written to %s\n", asmFile)

	// ========== 阶段 5: 汇编和链接 ==========
	fmt.Println("=== Assembling and Linking ===")
	exeFile := baseName + ".exe"

	if err := builder.BuildExecutable(asmCode, exeFile); err != nil {
		return fmt.Errorf("failed to build executable: %v", err)
	}

	fmt.Printf("Executable created: %s\n", exeFile)
	return nil
}

// compileWithLLVMBackend 使用 LLVM 后端编译
func compileWithLLVMBackend(program *parser.Program, analyzer *semantic.Analyzer, baseName string) error {
	// ========== 阶段 4: LLVM 代码生成 ==========
	fmt.Println("=== LLVM Code Generation ===")
	gen := codegen.New(analyzer)
	module, err := gen.Generate(program)
	if err != nil {
		return fmt.Errorf("code generation failed: %v", err)
	}

	// 写入 LLVM IR 到文件
	llFile := baseName + ".ll"
	llvmIR := module.String()
	if err := os.WriteFile(llFile, []byte(llvmIR), 0644); err != nil {
		return fmt.Errorf("failed to write LLVM IR: %v", err)
	}
	fmt.Printf("LLVM IR written to %s\n", llFile)

	// ========== 阶段 5: 后端编译 ==========
	// 使用 Clang 将 LLVM IR 编译为可执行文件
	fmt.Println("=== Compiling to executable ===")
	exeFile := baseName + ".exe"

	// 调用系统命令：clang hello.ll -o hello.exe
	cmd := exec.Command("clang", llFile, "-o", exeFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Clang output: %s\n", string(output))
		return fmt.Errorf("failed to compile with clang: %v", err)
	}

	fmt.Printf("Executable created: %s\n", exeFile)
	return nil
}
