// Package x64 provides executable file generation
package x64

import (
	"fmt"
	"goc/embedded"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ExecutableBuilder builds executable files from assembly code
type ExecutableBuilder struct {
	asm      *Assembler
	platform string // "windows" or "linux"
}

// NewExecutableBuilder creates a new executable builder
func NewExecutableBuilder(asm *Assembler) *ExecutableBuilder {
	return &ExecutableBuilder{
		asm:      asm,
		platform: runtime.GOOS,
	}
}

// GenerateNASM generates NASM-format assembly code
func (eb *ExecutableBuilder) GenerateNASM() string {
	var sb strings.Builder

	// Write format directive
	if eb.platform == "windows" {
		sb.WriteString("bits 64\n")
		sb.WriteString("default rel\n\n")
	} else {
		sb.WriteString("bits 64\n\n")
	}

	// Add exit syscall helper for Linux
	if eb.platform == "linux" {
		sb.WriteString("section .text\n")
		sb.WriteString("global _start\n\n")
	} else {
		sb.WriteString("section .text\n")
		sb.WriteString("global main\n\n")
	}

	// Convert assembly instructions to NASM syntax
	for _, code := range eb.asm.Code {
		switch c := code.(type) {
		case Section:
			sb.WriteString(c.String() + "\n")

		case Label:
			// Handle special labels
			if c.Name == "_start" && eb.platform == "windows" {
				// Skip _start on Windows, we use main
				continue
			}
			if c.Name == "main" && eb.platform == "linux" {
				// On Linux, we already have _start
				continue
			}
			sb.WriteString(c.String() + "\n")

		case Comment:
			sb.WriteString(c.String() + "\n")

		case Instruction:
			// Convert instruction to NASM syntax
			sb.WriteString("    ")
			sb.WriteString(eb.instructionToNASM(c))
			sb.WriteString("\n")
		}
	}

	// Add exit code for main function
	if eb.platform == "linux" {
		sb.WriteString("\n; Exit with return code from main\n")
		sb.WriteString("    mov rdi, rax    ; Return value is exit code\n")
		sb.WriteString("    mov rax, 60     ; syscall: exit\n")
		sb.WriteString("    syscall\n")
	}

	return sb.String()
}

// instructionToNASM converts an instruction to NASM syntax
func (eb *ExecutableBuilder) instructionToNASM(inst Instruction) string {
	// Special handling for AL register (8-bit)
	replaceAL := func(s string) string {
		// AL is used in SETcc instructions, keep it as is
		return s
	}

	if inst.Op1 == nil && inst.Op2 == nil {
		return inst.Opcode
	} else if inst.Op2 == nil {
		op1Str := replaceAL(inst.Op1.String())
		return fmt.Sprintf("%s %s", inst.Opcode, op1Str)
	}

	op1Str := replaceAL(inst.Op1.String())
	op2Str := replaceAL(inst.Op2.String())
	return fmt.Sprintf("%s %s, %s", inst.Opcode, op1Str, op2Str)
}

// BuildExecutable builds an executable file from assembly code
func (eb *ExecutableBuilder) BuildExecutable(asmCode, outputPath string) error {
	// Create temporary directory for intermediate files
	tempDir, err := os.MkdirTemp("", "goc-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write assembly file
	asmFile := filepath.Join(tempDir, "output.asm")
	if err := os.WriteFile(asmFile, []byte(asmCode), 0644); err != nil {
		return fmt.Errorf("failed to write assembly file: %w", err)
	}

	// Assemble with NASM
	objFile := filepath.Join(tempDir, "output.o")
	if err := eb.assemble(asmFile, objFile); err != nil {
		return fmt.Errorf("assembly failed: %w", err)
	}

	// Link
	if err := eb.link(objFile, outputPath); err != nil {
		return fmt.Errorf("linking failed: %w", err)
	}

	return nil
}

// assemble assembles the .asm file to .o file using NASM
func (eb *ExecutableBuilder) assemble(asmFile, objFile string) error {
	// 尝试获取嵌入的 NASM
	nasmPath, err := eb.getNASMPath()
	if err != nil {
		return err
	}

	var cmd *exec.Cmd

	if eb.platform == "windows" {
		// NASM for Windows: nasm -f win64 output.asm -o output.o
		cmd = exec.Command(nasmPath, "-f", "win64", asmFile, "-o", objFile)
	} else {
		// NASM for Linux: nasm -f elf64 output.asm -o output.o
		cmd = exec.Command(nasmPath, "-f", "elf64", asmFile, "-o", objFile)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("NASM failed: %w", err)
	}

	return nil
}

// getNASMPath 获取 NASM 的路径（优先使用嵌入版本）
func (eb *ExecutableBuilder) getNASMPath() (string, error) {
	// 首先尝试使用嵌入的 NASM
	if embedded.IsEmbedded() {
		path, err := embedded.GetNASMPath()
		if err == nil {
			fmt.Println("Using embedded NASM assembler")
			return path, nil
		}
		fmt.Printf("Warning: Failed to extract embedded NASM: %v\n", err)
	}

	// 回退到系统路径中的 NASM
	path, err := exec.LookPath("nasm")
	if err != nil {
		return "", fmt.Errorf("NASM not found. Please either:\n  1. Download NASM and place in embedded/tools/ directory, then rebuild\n  2. Install NASM system-wide (https://www.nasm.us/)")
	}

	fmt.Println("Using system NASM assembler")
	return path, nil
}

// link links the object file to create executable
func (eb *ExecutableBuilder) link(objFile, outputPath string) error {
	var cmd *exec.Cmd

	if eb.platform == "windows" {
		// Use link.exe (MSVC linker) or ld (MinGW)
		// Try MinGW ld first
		if _, err := exec.LookPath("ld"); err == nil {
			cmd = exec.Command("ld", objFile, "-o", outputPath,
				"-e", "main",
				"-subsystem", "console")
		} else if _, err := exec.LookPath("link"); err == nil {
			// MSVC linker
			cmd = exec.Command("link", objFile,
				fmt.Sprintf("/OUT:%s", outputPath),
				"/ENTRY:main",
				"/SUBSYSTEM:CONSOLE")
		} else {
			return fmt.Errorf("no suitable linker found (tried: ld, link.exe)")
		}
	} else {
		// Linux: use ld
		cmd = exec.Command("ld", objFile, "-o", outputPath)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("linker failed: %w", err)
	}

	return nil
}

// SaveAssembly saves the assembly code to a file
func (eb *ExecutableBuilder) SaveAssembly(asmCode, outputPath string) error {
	return os.WriteFile(outputPath, []byte(asmCode), 0644)
}
