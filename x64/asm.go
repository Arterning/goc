// Package x64 provides x86-64 assembly code generation and executable file generation.
package x64

import (
	"fmt"
)

// Register represents an x86-64 register
type Register int

const (
	// 通用寄存器 (64-bit)
	RAX Register = iota
	RBX
	RCX
	RDX
	RSI
	RDI
	RBP
	RSP
	R8
	R9
	R10
	R11
	R12
	R13
	R14
	R15

	// 32-bit 寄存器
	EAX
	EBX
	ECX
	EDX
	ESI
	EDI
	EBP
	ESP

	// 8-bit 寄存器 (低字节)
	AL  // RAX 的低 8 位
	BL  // RBX 的低 8 位
	CL  // RCX 的低 8 位
	DL  // RDX 的低 8 位

	// XMM 寄存器 (用于浮点运算)
	XMM0
	XMM1
	XMM2
	XMM3
	XMM4
	XMM5
	XMM6
	XMM7
	XMM8
	XMM9
	XMM10
	XMM11
	XMM12
	XMM13
	XMM14
	XMM15
)

// String returns the assembly name of the register
func (r Register) String() string {
	switch r {
	case RAX:
		return "rax"
	case RBX:
		return "rbx"
	case RCX:
		return "rcx"
	case RDX:
		return "rdx"
	case RSI:
		return "rsi"
	case RDI:
		return "rdi"
	case RBP:
		return "rbp"
	case RSP:
		return "rsp"
	case R8:
		return "r8"
	case R9:
		return "r9"
	case R10:
		return "r10"
	case R11:
		return "r11"
	case R12:
		return "r12"
	case R13:
		return "r13"
	case R14:
		return "r14"
	case R15:
		return "r15"
	case EAX:
		return "eax"
	case EBX:
		return "ebx"
	case ECX:
		return "ecx"
	case EDX:
		return "edx"
	case ESI:
		return "esi"
	case EDI:
		return "edi"
	case EBP:
		return "ebp"
	case ESP:
		return "esp"
	case AL:
		return "al"
	case BL:
		return "bl"
	case CL:
		return "cl"
	case DL:
		return "dl"
	case XMM0:
		return "xmm0"
	case XMM1:
		return "xmm1"
	case XMM2:
		return "xmm2"
	case XMM3:
		return "xmm3"
	case XMM4:
		return "xmm4"
	case XMM5:
		return "xmm5"
	case XMM6:
		return "xmm6"
	case XMM7:
		return "xmm7"
	case XMM8:
		return "xmm8"
	case XMM9:
		return "xmm9"
	case XMM10:
		return "xmm10"
	case XMM11:
		return "xmm11"
	case XMM12:
		return "xmm12"
	case XMM13:
		return "xmm13"
	case XMM14:
		return "xmm14"
	case XMM15:
		return "xmm15"
	default:
		return "unknown"
	}
}

// Operand represents an instruction operand
type Operand interface {
	String() string
	isOperand()
}

// RegOperand represents a register operand
type RegOperand struct {
	Reg Register
}

func (r RegOperand) String() string  { return r.Reg.String() }
func (r RegOperand) isOperand()      {}

// ImmOperand represents an immediate value operand
type ImmOperand struct {
	Value int64
}

func (i ImmOperand) String() string { return fmt.Sprintf("%d", i.Value) }
func (i ImmOperand) isOperand()     {}

// FloatImmOperand represents a floating-point immediate value
type FloatImmOperand struct {
	Value float32
}

func (f FloatImmOperand) String() string { return fmt.Sprintf("%f", f.Value) }
func (f FloatImmOperand) isOperand()     {}

// MemOperand represents a memory operand (e.g., [rbp-8])
type MemOperand struct {
	Base   Register
	Offset int64
}

func (m MemOperand) String() string {
	if m.Offset >= 0 {
		return fmt.Sprintf("[%s+%d]", m.Base.String(), m.Offset)
	}
	return fmt.Sprintf("[%s%d]", m.Base.String(), m.Offset)
}
func (m MemOperand) isOperand() {}

// LabelOperand represents a label operand (for jumps)
type LabelOperand struct {
	Label string
}

func (l LabelOperand) String() string { return l.Label }
func (l LabelOperand) isOperand()     {}

// Instruction represents an x86-64 instruction
type Instruction struct {
	Opcode string
	Op1    Operand
	Op2    Operand
}

// String returns the assembly representation of the instruction
func (i Instruction) String() string {
	if i.Op1 == nil && i.Op2 == nil {
		return i.Opcode
	} else if i.Op2 == nil {
		return fmt.Sprintf("%s %s", i.Opcode, i.Op1.String())
	}
	return fmt.Sprintf("%s %s, %s", i.Opcode, i.Op1.String(), i.Op2.String())
}

// Label represents an assembly label
type Label struct {
	Name string
}

func (l Label) String() string {
	return l.Name + ":"
}

// Comment represents an assembly comment
type Comment struct {
	Text string
}

func (c Comment) String() string {
	return "; " + c.Text
}

// Section represents an assembly section (.text, .data, etc.)
type Section struct {
	Name string
}

func (s Section) String() string {
	return "section " + s.Name
}

// Assembler provides helper methods to create instructions
type Assembler struct {
	Code []interface{} // Can be Instruction, Label, Comment, or Section
}

// NewAssembler creates a new assembler
func NewAssembler() *Assembler {
	return &Assembler{
		Code: make([]interface{}, 0),
	}
}

// Emit adds an instruction to the code
func (a *Assembler) Emit(opcode string, op1, op2 Operand) {
	a.Code = append(a.Code, Instruction{Opcode: opcode, Op1: op1, Op2: op2})
}

// EmitLabel adds a label to the code
func (a *Assembler) EmitLabel(name string) {
	a.Code = append(a.Code, Label{Name: name})
}

// EmitComment adds a comment to the code
func (a *Assembler) EmitComment(text string) {
	a.Code = append(a.Code, Comment{Text: text})
}

// EmitSection adds a section directive
func (a *Assembler) EmitSection(name string) {
	a.Code = append(a.Code, Section{Name: name})
}

// Helper methods for common operations

// Mov emits a MOV instruction
func (a *Assembler) Mov(dst, src Operand) {
	a.Emit("mov", dst, src)
}

// Movss emits a MOVSS instruction (move scalar single-precision float)
func (a *Assembler) Movss(dst, src Operand) {
	a.Emit("movss", dst, src)
}

// Push emits a PUSH instruction
func (a *Assembler) Push(src Operand) {
	a.Emit("push", src, nil)
}

// Pop emits a POP instruction
func (a *Assembler) Pop(dst Operand) {
	a.Emit("pop", dst, nil)
}

// Add emits an ADD instruction
func (a *Assembler) Add(dst, src Operand) {
	a.Emit("add", dst, src)
}

// Addss emits an ADDSS instruction (add scalar single-precision float)
func (a *Assembler) Addss(dst, src Operand) {
	a.Emit("addss", dst, src)
}

// Sub emits a SUB instruction
func (a *Assembler) Sub(dst, src Operand) {
	a.Emit("sub", dst, src)
}

// Subss emits a SUBSS instruction (subtract scalar single-precision float)
func (a *Assembler) Subss(dst, src Operand) {
	a.Emit("subss", dst, src)
}

// Imul emits an IMUL instruction (signed multiply)
func (a *Assembler) Imul(dst, src Operand) {
	a.Emit("imul", dst, src)
}

// Mulss emits a MULSS instruction (multiply scalar single-precision float)
func (a *Assembler) Mulss(dst, src Operand) {
	a.Emit("mulss", dst, src)
}

// Idiv emits an IDIV instruction (signed divide, result in RAX, remainder in RDX)
func (a *Assembler) Idiv(divisor Operand) {
	a.Emit("idiv", divisor, nil)
}

// Divss emits a DIVSS instruction (divide scalar single-precision float)
func (a *Assembler) Divss(dst, src Operand) {
	a.Emit("divss", dst, src)
}

// Cqo emits a CQO instruction (sign-extend RAX to RDX:RAX)
func (a *Assembler) Cqo() {
	a.Emit("cqo", nil, nil)
}

// Cmp emits a CMP instruction
func (a *Assembler) Cmp(op1, op2 Operand) {
	a.Emit("cmp", op1, op2)
}

// Ucomiss emits a UCOMISS instruction (unordered compare scalar single-precision float)
func (a *Assembler) Ucomiss(op1, op2 Operand) {
	a.Emit("ucomiss", op1, op2)
}

// Jmp emits an unconditional JMP instruction
func (a *Assembler) Jmp(label string) {
	a.Emit("jmp", LabelOperand{Label: label}, nil)
}

// Je emits a JE (jump if equal) instruction
func (a *Assembler) Je(label string) {
	a.Emit("je", LabelOperand{Label: label}, nil)
}

// Jne emits a JNE (jump if not equal) instruction
func (a *Assembler) Jne(label string) {
	a.Emit("jne", LabelOperand{Label: label}, nil)
}

// Jl emits a JL (jump if less) instruction
func (a *Assembler) Jl(label string) {
	a.Emit("jl", LabelOperand{Label: label}, nil)
}

// Jle emits a JLE (jump if less or equal) instruction
func (a *Assembler) Jle(label string) {
	a.Emit("jle", LabelOperand{Label: label}, nil)
}

// Jg emits a JG (jump if greater) instruction
func (a *Assembler) Jg(label string) {
	a.Emit("jg", LabelOperand{Label: label}, nil)
}

// Jge emits a JGE (jump if greater or equal) instruction
func (a *Assembler) Jge(label string) {
	a.Emit("jge", LabelOperand{Label: label}, nil)
}

// Ja emits a JA (jump if above, unsigned) instruction
func (a *Assembler) Ja(label string) {
	a.Emit("ja", LabelOperand{Label: label}, nil)
}

// Jb emits a JB (jump if below, unsigned) instruction
func (a *Assembler) Jb(label string) {
	a.Emit("jb", LabelOperand{Label: label}, nil)
}

// Call emits a CALL instruction
func (a *Assembler) Call(target string) {
	a.Emit("call", LabelOperand{Label: target}, nil)
}

// Ret emits a RET instruction
func (a *Assembler) Ret() {
	a.Emit("ret", nil, nil)
}

// Xor emits an XOR instruction
func (a *Assembler) Xor(dst, src Operand) {
	a.Emit("xor", dst, src)
}

// And emits an AND instruction
func (a *Assembler) And(dst, src Operand) {
	a.Emit("and", dst, src)
}

// Or emits an OR instruction
func (a *Assembler) Or(dst, src Operand) {
	a.Emit("or", dst, src)
}

// Test emits a TEST instruction
func (a *Assembler) Test(op1, op2 Operand) {
	a.Emit("test", op1, op2)
}

// Sete emits a SETE instruction (set byte if equal)
func (a *Assembler) Sete(dst Operand) {
	a.Emit("sete", dst, nil)
}

// Setne emits a SETNE instruction (set byte if not equal)
func (a *Assembler) Setne(dst Operand) {
	a.Emit("setne", dst, nil)
}

// Setl emits a SETL instruction (set byte if less)
func (a *Assembler) Setl(dst Operand) {
	a.Emit("setl", dst, nil)
}

// Setle emits a SETLE instruction (set byte if less or equal)
func (a *Assembler) Setle(dst Operand) {
	a.Emit("setle", dst, nil)
}

// Setg emits a SETG instruction (set byte if greater)
func (a *Assembler) Setg(dst Operand) {
	a.Emit("setg", dst, nil)
}

// Setge emits a SETGE instruction (set byte if greater or equal)
func (a *Assembler) Setge(dst Operand) {
	a.Emit("setge", dst, nil)
}

// Seta emits a SETA instruction (set byte if above, unsigned)
func (a *Assembler) Seta(dst Operand) {
	a.Emit("seta", dst, nil)
}

// Setb emits a SETB instruction (set byte if below, unsigned)
func (a *Assembler) Setb(dst Operand) {
	a.Emit("setb", dst, nil)
}

// Movzx emits a MOVZX instruction (move with zero-extend)
func (a *Assembler) Movzx(dst, src Operand) {
	a.Emit("movzx", dst, src)
}

// Neg emits a NEG instruction (negate)
func (a *Assembler) Neg(dst Operand) {
	a.Emit("neg", dst, nil)
}

// Not emits a NOT instruction (bitwise NOT)
func (a *Assembler) Not(dst Operand) {
	a.Emit("not", dst, nil)
}

// Cvtsi2ss emits a CVTSI2SS instruction (convert signed int to scalar single-precision float)
func (a *Assembler) Cvtsi2ss(dst, src Operand) {
	a.Emit("cvtsi2ss", dst, src)
}

// Cvttss2si emits a CVTTSS2SI instruction (convert scalar single-precision float to signed int with truncation)
func (a *Assembler) Cvttss2si(dst, src Operand) {
	a.Emit("cvttss2si", dst, src)
}
