// Package x64 provides ELF (Executable and Linkable Format) file generation for Linux
package x64

import (
	"encoding/binary"
	"os"
)

// ELFGenerator generates Linux ELF64 executable files
type ELFGenerator struct {
	code []byte
}

// NewELFGenerator creates a new ELF file generator
func NewELFGenerator(code []byte) *ELFGenerator {
	return &ELFGenerator{code: code}
}

// Generate generates an ELF executable file
func (elf *ELFGenerator) Generate(outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// ELF Constants
	const (
		loadAddress  = uint64(0x400000)
		codeOffset   = uint64(0x1000) // Start code at 4KB
		programHeader = uint64(64)     // Right after ELF header
	)

	entryPoint := loadAddress + codeOffset

	// Write ELF Header
	elf.writeELFHeader(file, entryPoint, programHeader)

	// Write Program Header
	elf.writeProgramHeader(file, loadAddress, codeOffset, uint64(len(elf.code)))

	// Pad to code offset
	currentPos, _ := file.Seek(0, 1)
	padSize := int64(codeOffset) - currentPos
	if padSize > 0 {
		file.Write(make([]byte, padSize))
	}

	// Write code
	file.Write(elf.code)

	// Make file executable
	file.Close()
	os.Chmod(outputPath, 0755)

	return nil
}

// writeELFHeader writes the ELF64 header
func (elf *ELFGenerator) writeELFHeader(file *os.File, entryPoint, phOffset uint64) {
	// ELF Magic number
	file.Write([]byte{0x7f, 'E', 'L', 'F'})

	// Class (64-bit)
	binary.Write(file, binary.LittleEndian, uint8(2))

	// Data (little-endian)
	binary.Write(file, binary.LittleEndian, uint8(1))

	// Version (current)
	binary.Write(file, binary.LittleEndian, uint8(1))

	// OS/ABI (System V)
	binary.Write(file, binary.LittleEndian, uint8(0))

	// ABI Version
	binary.Write(file, binary.LittleEndian, uint8(0))

	// Padding (7 bytes)
	file.Write(make([]byte, 7))

	// Type (executable)
	binary.Write(file, binary.LittleEndian, uint16(2))

	// Machine (x86-64)
	binary.Write(file, binary.LittleEndian, uint16(0x3E))

	// Version
	binary.Write(file, binary.LittleEndian, uint32(1))

	// Entry point address
	binary.Write(file, binary.LittleEndian, entryPoint)

	// Program header offset
	binary.Write(file, binary.LittleEndian, phOffset)

	// Section header offset (0 = no section headers)
	binary.Write(file, binary.LittleEndian, uint64(0))

	// Flags
	binary.Write(file, binary.LittleEndian, uint32(0))

	// ELF header size
	binary.Write(file, binary.LittleEndian, uint16(64))

	// Program header entry size
	binary.Write(file, binary.LittleEndian, uint16(56))

	// Number of program headers
	binary.Write(file, binary.LittleEndian, uint16(1))

	// Section header entry size
	binary.Write(file, binary.LittleEndian, uint16(0))

	// Number of section headers
	binary.Write(file, binary.LittleEndian, uint16(0))

	// Section header string table index
	binary.Write(file, binary.LittleEndian, uint16(0))
}

// writeProgramHeader writes a program header (LOAD segment)
func (elf *ELFGenerator) writeProgramHeader(file *os.File, loadAddr, offset, size uint64) {
	// Type (PT_LOAD = 1)
	binary.Write(file, binary.LittleEndian, uint32(1))

	// Flags (PF_X | PF_R = executable + readable = 5)
	binary.Write(file, binary.LittleEndian, uint32(5))

	// Offset in file
	binary.Write(file, binary.LittleEndian, offset)

	// Virtual address
	binary.Write(file, binary.LittleEndian, loadAddr+offset)

	// Physical address (same as virtual)
	binary.Write(file, binary.LittleEndian, loadAddr+offset)

	// Size in file
	binary.Write(file, binary.LittleEndian, size)

	// Size in memory (same as file size)
	binary.Write(file, binary.LittleEndian, size)

	// Alignment (page size = 4096)
	binary.Write(file, binary.LittleEndian, uint64(0x1000))
}
