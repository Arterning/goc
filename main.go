package main

import (
	"fmt"
	"goc/codegen"
	"goc/lexer"
	"goc/parser"
	"goc/semantic"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: goc <source-file>")
		fmt.Println("Example: goc program.goc")
		os.Exit(1)
	}

	sourceFile := os.Args[1]

	// Read source file
	source, err := os.ReadFile(sourceFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Compile
	if err := compile(string(source), sourceFile); err != nil {
		fmt.Printf("Compilation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Compilation successful!")
}

func compile(source string, sourceFile string) error {
	// Lexical analysis
	fmt.Println("=== Lexical Analysis ===")
	lex := lexer.New(source)

	// Syntax analysis
	fmt.Println("=== Syntax Analysis ===")
	p := parser.New(lex)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("Parser errors:")
		for _, err := range p.Errors() {
			fmt.Printf("  %s\n", err)
		}
		return fmt.Errorf("parsing failed")
	}
	fmt.Println("Parsing successful")

	// Semantic analysis
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

	// Code generation
	fmt.Println("=== Code Generation ===")
	gen := codegen.New(analyzer)
	module, err := gen.Generate(program)
	if err != nil {
		return fmt.Errorf("code generation failed: %v", err)
	}

	// Write LLVM IR to file
	baseName := strings.TrimSuffix(sourceFile, filepath.Ext(sourceFile))
	llFile := baseName + ".ll"

	llvmIR := module.String()
	if err := os.WriteFile(llFile, []byte(llvmIR), 0644); err != nil {
		return fmt.Errorf("failed to write LLVM IR: %v", err)
	}
	fmt.Printf("LLVM IR written to %s\n", llFile)

	// Compile LLVM IR to object file using clang
	fmt.Println("=== Compiling to executable ===")
	objFile := baseName + ".exe"

	cmd := exec.Command("clang", llFile, "-o", objFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Clang output: %s\n", string(output))
		return fmt.Errorf("failed to compile with clang: %v", err)
	}

	fmt.Printf("Executable created: %s\n", objFile)
	return nil
}
