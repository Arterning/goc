// Package x64 provides PE (Portable Executable) file generation for Windows
package x64

import (
	"encoding/binary"
	"os"
)

// PEGenerator generates Windows PE32+ executable files
type PEGenerator struct {
	code []byte
}

// NewPEGenerator creates a new PE file generator
func NewPEGenerator(code []byte) *PEGenerator {
	return &PEGenerator{code: code}
}

// Generate generates a PE executable file
func (pe *PEGenerator) Generate(outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Calculate sizes
	codeSize := uint32(len(pe.code))
	codeSize = align(codeSize, 512) // File alignment

	// Base addresses
	imageBase := uint64(0x140000000) // Default for x64
	codeRVA := uint32(0x1000)        // Section starts at 4KB
	entryPointRVA := codeRVA         // Entry point is at start of code

	// Write DOS Header
	pe.writeDOSHeader(file)

	// Write PE Signature
	binary.Write(file, binary.LittleEndian, uint32(0x00004550)) // "PE\0\0"

	// Write COFF File Header
	pe.writeCOFFHeader(file, 1) // 1 section

	// Write Optional Header (PE32+)
	pe.writeOptionalHeader(file, imageBase, entryPointRVA, codeSize, codeRVA)

	// Write Section Table
	pe.writeSectionHeader(file, ".text", codeSize, codeRVA)

	// Pad to section alignment
	currentPos, _ := file.Seek(0, 1)
	padSize := int64(512) - (currentPos % 512)
	if padSize < 512 {
		file.Write(make([]byte, padSize))
	}

	// Write code section
	file.Write(pe.code)

	// Pad code to file alignment
	if remainder := len(pe.code) % 512; remainder != 0 {
		file.Write(make([]byte, 512-remainder))
	}

	return nil
}

// writeDOSHeader writes the DOS header and stub
func (pe *PEGenerator) writeDOSHeader(file *os.File) {
	// DOS Header (64 bytes)
	dosHeader := make([]byte, 64)

	// MZ signature
	dosHeader[0] = 'M'
	dosHeader[1] = 'Z'

	// Bytes on last page
	binary.LittleEndian.PutUint16(dosHeader[2:], 0x90)

	// Pages in file
	binary.LittleEndian.PutUint16(dosHeader[4:], 0x03)

	// Paragraph in header
	binary.LittleEndian.PutUint16(dosHeader[8:], 0x04)

	// Maximum extra paragraphs
	binary.LittleEndian.PutUint16(dosHeader[10:], 0xFFFF)

	// Initial SP
	binary.LittleEndian.PutUint16(dosHeader[16:], 0xB8)

	// File address of PE header
	binary.LittleEndian.PutUint32(dosHeader[60:], 0x80)

	file.Write(dosHeader)

	// DOS Stub (simple "This program cannot be run in DOS mode" message)
	dosStub := []byte{
		// 14 bytes: push cs; pop ds; mov dx, 0x0E; mov ah, 0x09; int 0x21;
		0x0E, 0x1F, 0xBA, 0x0E, 0x00, 0xB4, 0x09, 0xCD,
		0x21, 0xB8, 0x01, 0x4C, 0xCD, 0x21,
		// Message
		'T', 'h', 'i', 's', ' ', 'p', 'r', 'o', 'g', 'r', 'a', 'm', ' ',
		'c', 'a', 'n', 'n', 'o', 't', ' ', 'b', 'e', ' ', 'r', 'u', 'n', ' ',
		'i', 'n', ' ', 'D', 'O', 'S', ' ', 'm', 'o', 'd', 'e', '.', '\r', '\r', '\n', '$',
		0, 0, 0, 0, 0, 0, 0, 0,
	}

	file.Write(dosStub)

	// Pad to offset 0x80 (128 bytes total)
	currentSize := 64 + len(dosStub)
	if currentSize < 128 {
		file.Write(make([]byte, 128-currentSize))
	}
}

// writeCOFFHeader writes the COFF file header
func (pe *PEGenerator) writeCOFFHeader(file *os.File, numSections uint16) {
	// Machine type: x64
	binary.Write(file, binary.LittleEndian, uint16(0x8664))

	// Number of sections
	binary.Write(file, binary.LittleEndian, numSections)

	// TimeDateStamp (set to 0)
	binary.Write(file, binary.LittleEndian, uint32(0))

	// PointerToSymbolTable (0 = no symbols)
	binary.Write(file, binary.LittleEndian, uint32(0))

	// NumberOfSymbols
	binary.Write(file, binary.LittleEndian, uint32(0))

	// SizeOfOptionalHeader (for PE32+ it's 240 bytes)
	binary.Write(file, binary.LittleEndian, uint16(240))

	// Characteristics (executable, no relocations, 64-bit)
	binary.Write(file, binary.LittleEndian, uint16(0x0022))
}

// writeOptionalHeader writes the PE32+ optional header
func (pe *PEGenerator) writeOptionalHeader(file *os.File, imageBase uint64, entryPoint, codeSize, codeRVA uint32) {
	// Magic (PE32+ = 0x20B)
	binary.Write(file, binary.LittleEndian, uint16(0x020B))

	// Linker version
	binary.Write(file, binary.LittleEndian, uint8(14))
	binary.Write(file, binary.LittleEndian, uint8(0))

	// SizeOfCode
	binary.Write(file, binary.LittleEndian, codeSize)

	// SizeOfInitializedData
	binary.Write(file, binary.LittleEndian, uint32(0))

	// SizeOfUninitializedData
	binary.Write(file, binary.LittleEndian, uint32(0))

	// AddressOfEntryPoint
	binary.Write(file, binary.LittleEndian, entryPoint)

	// BaseOfCode
	binary.Write(file, binary.LittleEndian, codeRVA)

	// ImageBase
	binary.Write(file, binary.LittleEndian, imageBase)

	// SectionAlignment
	binary.Write(file, binary.LittleEndian, uint32(0x1000))

	// FileAlignment
	binary.Write(file, binary.LittleEndian, uint32(0x200))

	// OS Version (6.0 = Vista/Server 2008)
	binary.Write(file, binary.LittleEndian, uint16(6))
	binary.Write(file, binary.LittleEndian, uint16(0))

	// Image Version
	binary.Write(file, binary.LittleEndian, uint16(0))
	binary.Write(file, binary.LittleEndian, uint16(0))

	// Subsystem Version (6.0)
	binary.Write(file, binary.LittleEndian, uint16(6))
	binary.Write(file, binary.LittleEndian, uint16(0))

	// Win32VersionValue (reserved, must be 0)
	binary.Write(file, binary.LittleEndian, uint32(0))

	// SizeOfImage
	sizeOfImage := align(codeRVA+codeSize, 0x1000)
	binary.Write(file, binary.LittleEndian, sizeOfImage)

	// SizeOfHeaders
	binary.Write(file, binary.LittleEndian, uint32(0x200))

	// CheckSum (0 for non-driver executables)
	binary.Write(file, binary.LittleEndian, uint32(0))

	// Subsystem (3 = console)
	binary.Write(file, binary.LittleEndian, uint16(3))

	// DLL Characteristics
	binary.Write(file, binary.LittleEndian, uint16(0x8160))

	// Stack Reserve/Commit
	binary.Write(file, binary.LittleEndian, uint64(0x100000))
	binary.Write(file, binary.LittleEndian, uint64(0x1000))

	// Heap Reserve/Commit
	binary.Write(file, binary.LittleEndian, uint64(0x100000))
	binary.Write(file, binary.LittleEndian, uint64(0x1000))

	// LoaderFlags (reserved, must be 0)
	binary.Write(file, binary.LittleEndian, uint32(0))

	// NumberOfRvaAndSizes (16 is standard)
	binary.Write(file, binary.LittleEndian, uint32(16))

	// Data Directories (16 entries, 8 bytes each = 128 bytes)
	// We're not using any data directories, so fill with zeros
	file.Write(make([]byte, 128))
}

// writeSectionHeader writes a section header
func (pe *PEGenerator) writeSectionHeader(file *os.File, name string, size, rva uint32) {
	// Name (8 bytes, null-padded)
	nameBytes := make([]byte, 8)
	copy(nameBytes, name)
	file.Write(nameBytes)

	// VirtualSize
	binary.Write(file, binary.LittleEndian, size)

	// VirtualAddress (RVA)
	binary.Write(file, binary.LittleEndian, rva)

	// SizeOfRawData (file size)
	rawSize := align(size, 512)
	binary.Write(file, binary.LittleEndian, rawSize)

	// PointerToRawData (file offset)
	binary.Write(file, binary.LittleEndian, uint32(0x200))

	// PointerToRelocations
	binary.Write(file, binary.LittleEndian, uint32(0))

	// PointerToLineNumbers
	binary.Write(file, binary.LittleEndian, uint32(0))

	// NumberOfRelocations
	binary.Write(file, binary.LittleEndian, uint16(0))

	// NumberOfLineNumbers
	binary.Write(file, binary.LittleEndian, uint16(0))

	// Characteristics (code, executable, readable)
	binary.Write(file, binary.LittleEndian, uint32(0x60000020))
}

// align rounds up to the next multiple of alignment
func align(value, alignment uint32) uint32 {
	if remainder := value % alignment; remainder != 0 {
		return value + (alignment - remainder)
	}
	return value
}
