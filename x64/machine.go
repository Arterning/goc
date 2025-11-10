// Package x64 provides direct x86-64 machine code generation
package x64

import (
	"fmt"
)

// MachineCodeGenerator generates x86-64 machine code directly
type MachineCodeGenerator struct {
	code      []byte            // Generated machine code
	labels    map[string]int    // Label name -> code offset
	fixups    []Fixup           // Relocations to fix up later
}

// Fixup represents a location that needs to be fixed up after all code is generated
type Fixup struct {
	Offset int    // Offset in code where fixup is needed
	Label  string // Target label name
	Type   FixupType
}

// FixupType represents the type of fixup needed
type FixupType int

const (
	FixupRel32 FixupType = iota // 32-bit relative offset
	FixupRel8                    // 8-bit relative offset
)

// NewMachineCodeGenerator creates a new machine code generator
func NewMachineCodeGenerator() *MachineCodeGenerator {
	return &MachineCodeGenerator{
		code:   make([]byte, 0, 4096),
		labels: make(map[string]int),
		fixups: make([]Fixup, 0),
	}
}

// GetCode returns the generated machine code
func (m *MachineCodeGenerator) GetCode() []byte {
	return m.code
}

// Emit emits raw bytes
func (m *MachineCodeGenerator) Emit(bytes ...byte) {
	m.code = append(m.code, bytes...)
}

// EmitLabel marks a label at the current position
func (m *MachineCodeGenerator) EmitLabel(name string) {
	m.labels[name] = len(m.code)
}

// CurrentOffset returns the current code offset
func (m *MachineCodeGenerator) CurrentOffset() int {
	return len(m.code)
}

// ========== x86-64 Instruction Encoding ==========

// ModRM encodes a ModR/M byte
// Format: [mod(2) | reg(3) | rm(3)]
func ModRM(mod, reg, rm byte) byte {
	return (mod << 6) | ((reg & 0x7) << 3) | (rm & 0x7)
}

// SIB encodes a SIB byte
// Format: [scale(2) | index(3) | base(3)]
func SIB(scale, index, base byte) byte {
	return (scale << 6) | ((index & 0x7) << 3) | (base & 0x7)
}

// REX prefix for 64-bit operations
// Format: 0100WRXB
// W: 1 for 64-bit operand
// R: Extension of ModRM.reg
// X: Extension of SIB.index
// B: Extension of ModRM.rm or SIB.base
func REX(w, r, x, b byte) byte {
	return 0x40 | (w << 3) | (r << 2) | (x << 1) | b
}

// ========== Common Instructions ==========

// EmitPushR64 emits PUSH r64
func (m *MachineCodeGenerator) EmitPushR64(reg byte) {
	// PUSH r64: 50+rd
	m.Emit(0x50 + (reg & 0x7))
}

// EmitPopR64 emits POP r64
func (m *MachineCodeGenerator) EmitPopR64(reg byte) {
	// POP r64: 58+rd
	m.Emit(0x58 + (reg & 0x7))
}

// EmitRet emits RET
func (m *MachineCodeGenerator) EmitRet() {
	// RET: C3
	m.Emit(0xC3)
}

// EmitMovR64Imm32 emits MOV r64, imm32 (sign-extended)
func (m *MachineCodeGenerator) EmitMovR64Imm32(reg byte, imm int32) {
	// MOV r64, imm32: REX.W + C7 /0 id
	m.Emit(REX(1, 0, 0, 0))
	m.Emit(0xC7)
	m.Emit(ModRM(3, 0, reg)) // mod=11 (register), reg=0, rm=reg
	m.EmitInt32(imm)
}

// EmitMovR32Imm32 emits MOV r32, imm32
func (m *MachineCodeGenerator) EmitMovR32Imm32(reg byte, imm int32) {
	// MOV r32, imm32: B8+rd id
	m.Emit(0xB8 + (reg & 0x7))
	m.EmitInt32(imm)
}

// EmitMovR64R64 emits MOV r64, r64
func (m *MachineCodeGenerator) EmitMovR64R64(dst, src byte) {
	// MOV r64, r64: REX.W + 89 /r
	m.Emit(REX(1, 0, 0, 0))
	m.Emit(0x89)
	m.Emit(ModRM(3, src, dst))
}

// EmitMovR32R32 emits MOV r32, r32
func (m *MachineCodeGenerator) EmitMovR32R32(dst, src byte) {
	// MOV r32, r32: 89 /r
	m.Emit(0x89)
	m.Emit(ModRM(3, src, dst))
}

// EmitMovR32Mem emits MOV r32, [base + offset]
func (m *MachineCodeGenerator) EmitMovR32Mem(reg, base byte, offset int32) {
	// MOV r32, m32: 8B /r
	m.Emit(0x8B)

	if offset == 0 && base != 5 { // RBP needs displacement
		// [base]
		m.Emit(ModRM(0, reg, base))
		if base == 4 { // RSP needs SIB
			m.Emit(SIB(0, 4, 4))
		}
	} else if offset >= -128 && offset < 128 {
		// [base + disp8]
		m.Emit(ModRM(1, reg, base))
		if base == 4 { // RSP needs SIB
			m.Emit(SIB(0, 4, 4))
		}
		m.Emit(byte(offset))
	} else {
		// [base + disp32]
		m.Emit(ModRM(2, reg, base))
		if base == 4 { // RSP needs SIB
			m.Emit(SIB(0, 4, 4))
		}
		m.EmitInt32(offset)
	}
}

// EmitMovMemR32 emits MOV [base + offset], r32
func (m *MachineCodeGenerator) EmitMovMemR32(base byte, offset int32, reg byte) {
	// MOV m32, r32: 89 /r
	m.Emit(0x89)

	if offset == 0 && base != 5 { // RBP needs displacement
		// [base]
		m.Emit(ModRM(0, reg, base))
		if base == 4 { // RSP needs SIB
			m.Emit(SIB(0, 4, 4))
		}
	} else if offset >= -128 && offset < 128 {
		// [base + disp8]
		m.Emit(ModRM(1, reg, base))
		if base == 4 { // RSP needs SIB
			m.Emit(SIB(0, 4, 4))
		}
		m.Emit(byte(offset))
	} else {
		// [base + disp32]
		m.Emit(ModRM(2, reg, base))
		if base == 4 { // RSP needs SIB
			m.Emit(SIB(0, 4, 4))
		}
		m.EmitInt32(offset)
	}
}

// EmitSubR64Imm emits SUB r64, imm32
func (m *MachineCodeGenerator) EmitSubR64Imm(reg byte, imm int32) {
	// SUB r64, imm32: REX.W + 81 /5 id
	m.Emit(REX(1, 0, 0, 0))
	if imm >= -128 && imm < 128 {
		// Use sign-extended imm8
		m.Emit(0x83)
		m.Emit(ModRM(3, 5, reg))
		m.Emit(byte(imm))
	} else {
		m.Emit(0x81)
		m.Emit(ModRM(3, 5, reg))
		m.EmitInt32(imm)
	}
}

// EmitAddR64Imm emits ADD r64, imm32
func (m *MachineCodeGenerator) EmitAddR64Imm(reg byte, imm int32) {
	// ADD r64, imm32: REX.W + 81 /0 id
	m.Emit(REX(1, 0, 0, 0))
	if imm >= -128 && imm < 128 {
		// Use sign-extended imm8
		m.Emit(0x83)
		m.Emit(ModRM(3, 0, reg))
		m.Emit(byte(imm))
	} else {
		m.Emit(0x81)
		m.Emit(ModRM(3, 0, reg))
		m.EmitInt32(imm)
	}
}

// EmitAddR32R32 emits ADD r32, r32
func (m *MachineCodeGenerator) EmitAddR32R32(dst, src byte) {
	// ADD r32, r32: 01 /r
	m.Emit(0x01)
	m.Emit(ModRM(3, src, dst))
}

// EmitSubR32R32 emits SUB r32, r32
func (m *MachineCodeGenerator) EmitSubR32R32(dst, src byte) {
	// SUB r32, r32: 29 /r
	m.Emit(0x29)
	m.Emit(ModRM(3, src, dst))
}

// EmitNegR32 emits NEG r32
func (m *MachineCodeGenerator) EmitNegR32(reg byte) {
	// NEG r32: F7 /3
	m.Emit(0xF7)
	m.Emit(ModRM(3, 3, reg))
}

// EmitJmp emits unconditional jump to label
func (m *MachineCodeGenerator) EmitJmp(label string) {
	// JMP rel32: E9 cd
	m.Emit(0xE9)
	m.fixups = append(m.fixups, Fixup{
		Offset: len(m.code),
		Label:  label,
		Type:   FixupRel32,
	})
	m.EmitInt32(0) // Placeholder
}

// EmitJe emits JE (jump if equal) to label
func (m *MachineCodeGenerator) EmitJe(label string) {
	// JE rel32: 0F 84 cd
	m.Emit(0x0F, 0x84)
	m.fixups = append(m.fixups, Fixup{
		Offset: len(m.code),
		Label:  label,
		Type:   FixupRel32,
	})
	m.EmitInt32(0) // Placeholder
}

// EmitTestR32R32 emits TEST r32, r32
func (m *MachineCodeGenerator) EmitTestR32R32(reg1, reg2 byte) {
	// TEST r32, r32: 85 /r
	m.Emit(0x85)
	m.Emit(ModRM(3, reg2, reg1))
}

// EmitCmpR32R32 emits CMP r32, r32
func (m *MachineCodeGenerator) EmitCmpR32R32(reg1, reg2 byte) {
	// CMP r32, r32: 39 /r
	m.Emit(0x39)
	m.Emit(ModRM(3, reg2, reg1))
}

// EmitCall emits CALL to label
func (m *MachineCodeGenerator) EmitCall(label string) {
	// CALL rel32: E8 cd
	m.Emit(0xE8)
	m.fixups = append(m.fixups, Fixup{
		Offset: len(m.code),
		Label:  label,
		Type:   FixupRel32,
	})
	m.EmitInt32(0) // Placeholder
}

// ========== Helper Methods ==========

// EmitInt32 emits a 32-bit integer in little-endian
func (m *MachineCodeGenerator) EmitInt32(value int32) {
	m.Emit(byte(value), byte(value>>8), byte(value>>16), byte(value>>24))
}

// EmitInt64 emits a 64-bit integer in little-endian
func (m *MachineCodeGenerator) EmitInt64(value int64) {
	m.EmitInt32(int32(value))
	m.EmitInt32(int32(value >> 32))
}

// ApplyFixups applies all fixups to resolve label references
func (m *MachineCodeGenerator) ApplyFixups() error {
	for _, fixup := range m.fixups {
		targetOffset, ok := m.labels[fixup.Label]
		if !ok {
			return fmt.Errorf("undefined label: %s", fixup.Label)
		}

		switch fixup.Type {
		case FixupRel32:
			// Calculate relative offset
			// offset = target - (fixup_position + 4)
			rel := int32(targetOffset - (fixup.Offset + 4))
			m.code[fixup.Offset] = byte(rel)
			m.code[fixup.Offset+1] = byte(rel >> 8)
			m.code[fixup.Offset+2] = byte(rel >> 16)
			m.code[fixup.Offset+3] = byte(rel >> 24)

		case FixupRel8:
			rel := int32(targetOffset - (fixup.Offset + 1))
			if rel < -128 || rel > 127 {
				return fmt.Errorf("relative offset too large for 8-bit: %d", rel)
			}
			m.code[fixup.Offset] = byte(rel)
		}
	}
	return nil
}

// Register encoding constants
const (
	RegRAX byte = 0
	RegRCX byte = 1
	RegRDX byte = 2
	RegRBX byte = 3
	RegRSP byte = 4
	RegRBP byte = 5
	RegRSI byte = 6
	RegRDI byte = 7
)
