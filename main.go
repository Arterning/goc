// Package main 是 GoC 编译器的入口点
// GoC 是一个类 C 语言编译器，使用 Go 语言编写，LLVM 作为后端
//
// 编译流程：
// 源代码 -> 词法分析 -> 语法分析 -> 语义分析 -> 代码生成 -> LLVM IR -> Zig -> 可执行文件
//
// 使用方法：
//   goc <source-file>
//   例如：goc examples/hello.goc
//
// 输出：
//   - <source-file>.ll    LLVM IR 中间代码
//   - <source-file>.exe   可执行文件
package main

import (
	"fmt"
	"goc/backend"
	"goc/codegen"
	"goc/lexer"
	"goc/parser"
	"goc/semantic"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// main 编译器主函数
// 负责：
// 1. 解析命令行参数
// 2. 读取源文件
// 3. 调用编译流程
// 4. 处理错误
func main() {
	// 检查命令行参数
	if len(os.Args) < 2 {
		fmt.Println("Usage: goc <source-file> [source-file2] ...")
		fmt.Println("Example: goc program.goc")
		fmt.Println("         goc main.goc utils.goc math.goc")
		os.Exit(1)
	}

	sourceFiles := os.Args[1:]

	// 检查是否为多文件编译
	if len(sourceFiles) == 1 {
		// 单文件编译（保持原有行为）
		sourceFile := sourceFiles[0]
		source, err := os.ReadFile(sourceFile)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			os.Exit(1)
		}

		if err := compile(string(source), sourceFile); err != nil {
			fmt.Printf("Compilation failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		// 多文件编译
		if err := compileMultipleFiles(sourceFiles); err != nil {
			fmt.Printf("Compilation failed: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("Compilation successful!")
}

// compile 编译源代码的核心函数
// 参数 source: 源代码字符串
// 参数 sourceFile: 源文件路径（用于生成输出文件名）
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
//    输出：LLVM IR
//    功能：将高级语法转换为中间表示
//
// 5. 后端编译（Backend Compilation）
//    输入：LLVM IR
//    输出：可执行文件
//    工具：Zig（嵌入式 LLVM 编译器，支持交叉编译）
func compile(source string, sourceFile string) error {
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

	// ========== 阶段 4: 代码生成 ==========
	fmt.Println("=== Code Generation ===")
	gen := codegen.New(analyzer)
	module, err := gen.Generate(program)
	if err != nil {
		return fmt.Errorf("code generation failed: %v", err)
	}

	// 写入 LLVM IR 到文件
	// 例如：hello.goc -> hello.ll
	baseName := strings.TrimSuffix(sourceFile, filepath.Ext(sourceFile))
	llFile := baseName + ".ll"

	llvmIR := module.String()
	if err := os.WriteFile(llFile, []byte(llvmIR), 0644); err != nil {
		return fmt.Errorf("failed to write LLVM IR: %v", err)
	}
	fmt.Printf("LLVM IR written to %s\n", llFile)

	// ========== 阶段 5: 后端编译 ==========
	// 使用 Zig 将 LLVM IR 编译为可执行文件
	fmt.Println("=== Compiling to executable ===")

	// 创建 Zig 编译器实例
	zigCompiler, err := backend.NewZigCompiler()
	if err != nil {
		return fmt.Errorf("failed to initialize zig compiler: %v", err)
	}

	// 确定输出文件名（根据平台添加合适的扩展名）
	objFile := baseName
	if runtime.GOOS == "windows" {
		objFile += ".exe"
	}

	// 编译选项
	opts := backend.CompileOptions{
		InputFile:  llFile,
		OutputFile: objFile,
		TargetOS:   runtime.GOOS,   // 当前操作系统
		TargetArch: runtime.GOARCH, // 当前架构
		Optimize:   true,           // 启用优化
	}

	// 使用 zig cc 编译 LLVM IR
	if err := zigCompiler.Compile(opts); err != nil {
		return fmt.Errorf("zig compilation failed: %v", err)
	}

	fmt.Printf("Executable created: %s\n", objFile)
	return nil
}

// compileMultipleFiles 编译多个源文件并链接
// 参数 sourceFiles: 源文件路径列表
// 返回值: 编译错误（如果有）
//
// 编译流程：
// 1. 解析所有文件，收集 AST
// 2. 全局语义分析（收集所有函数声明）
// 3. 为每个文件生成 LLVM IR
// 4. 使用 zig cc 将所有 LLVM IR 文件链接成一个可执行文件
func compileMultipleFiles(sourceFiles []string) error {
	fmt.Println("=== Multi-file Compilation ===")
	fmt.Printf("Compiling %d files...\n", len(sourceFiles))

	// 存储每个文件的 AST 和文件名
	type FileInfo struct {
		fileName string
		program  *parser.Program
	}
	var fileInfos []FileInfo

	// 阶段 1: 解析所有文件
	fmt.Println("\n=== Phase 1: Parsing ===")
	for _, sourceFile := range sourceFiles {
		fmt.Printf("Parsing %s...\n", sourceFile)

		// 读取源文件
		source, err := os.ReadFile(sourceFile)
		if err != nil {
			return fmt.Errorf("error reading file %s: %v", sourceFile, err)
		}

		// 词法分析
		lex := lexer.New(string(source))

		// 语法分析
		p := parser.New(lex)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			fmt.Printf("Parser errors in %s:\n", sourceFile)
			for _, err := range p.Errors() {
				fmt.Printf("  %s\n", err)
			}
			return fmt.Errorf("parsing failed for %s", sourceFile)
		}

		fileInfos = append(fileInfos, FileInfo{
			fileName: sourceFile,
			program:  program,
		})
	}
	fmt.Println("All files parsed successfully")

	// 阶段 2: 全局语义分析
	fmt.Println("\n=== Phase 2: Semantic Analysis ===")

	// 创建全局语义分析器
	globalAnalyzer := semantic.New()

	// 第一遍：收集所有文件的函数声明
	fmt.Println("Collecting function declarations...")
	for _, info := range fileInfos {
		for _, fn := range info.program.Functions {
			globalAnalyzer.DeclareFunctionFromExternal(fn)
		}
	}

	// 第二遍：分析每个文件的函数体
	fmt.Println("Analyzing function bodies...")
	for _, info := range fileInfos {
		fmt.Printf("  Analyzing %s...\n", info.fileName)
		if !globalAnalyzer.AnalyzeFunctions(info.program) {
			fmt.Printf("Semantic errors in %s:\n", info.fileName)
			for _, err := range globalAnalyzer.Errors() {
				fmt.Printf("  %s\n", err)
			}
			return fmt.Errorf("semantic analysis failed for %s", info.fileName)
		}
	}
	fmt.Println("Semantic analysis successful")

	// 阶段 3: 代码生成
	fmt.Println("\n=== Phase 3: Code Generation ===")

	// 合并所有程序的函数为一个大程序
	mergedProgram := &parser.Program{
		Functions: []*parser.FunctionDecl{},
	}
	for _, info := range fileInfos {
		mergedProgram.Functions = append(mergedProgram.Functions, info.program.Functions...)
	}

	// 为合并的程序生成一个 LLVM IR 模块
	fmt.Println("Generating unified LLVM IR module...")
	gen := codegen.New(globalAnalyzer)
	module, err := gen.Generate(mergedProgram)
	if err != nil {
		return fmt.Errorf("code generation failed: %v", err)
	}

	// 写入单个 LLVM IR 文件
	baseName := strings.TrimSuffix(sourceFiles[0], filepath.Ext(sourceFiles[0]))
	llFile := baseName + ".ll"

	llvmIR := module.String()
	if err := os.WriteFile(llFile, []byte(llvmIR), 0644); err != nil {
		return fmt.Errorf("failed to write LLVM IR: %v", err)
	}
	fmt.Printf("LLVM IR written to %s\n", llFile)

	// 阶段 4: 编译为可执行文件
	fmt.Println("\n=== Compiling to executable ===")

	// 创建 Zig 编译器实例
	zigCompiler, err := backend.NewZigCompiler()
	if err != nil {
		return fmt.Errorf("failed to initialize zig compiler: %v", err)
	}

	// 确定输出文件名（使用第一个源文件的名字）
	outputFile := baseName
	if runtime.GOOS == "windows" {
		outputFile += ".exe"
	}

	// 编译选项
	opts := backend.CompileOptions{
		InputFile:  llFile,
		OutputFile: outputFile,
		TargetOS:   runtime.GOOS,
		TargetArch: runtime.GOARCH,
		Optimize:   true,
	}

	// 使用 zig cc 编译
	if err := zigCompiler.Compile(opts); err != nil {
		return fmt.Errorf("compilation failed: %v", err)
	}

	fmt.Printf("Executable created: %s\n", outputFile)
	return nil
}
