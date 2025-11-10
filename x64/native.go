// Package x64 provides native machine code generation
package x64

import (
	"fmt"
	"goc/parser"
	"goc/semantic"
	"goc/types"
	"runtime"
)

// NativeCodeGenerator generates native machine code directly
type NativeCodeGenerator struct {
	machine      *MachineCodeGenerator
	analyzer     *semantic.Analyzer
	varOffsets   map[string]int32 // Variable name -> RBP offset
	stackOffset  int32            // Current stack offset from RBP
	labelCounter int              // For generating unique labels
}

// NewNativeCodeGenerator creates a new native code generator
func NewNativeCodeGenerator(analyzer *semantic.Analyzer) *NativeCodeGenerator {
	return &NativeCodeGenerator{
		machine:     NewMachineCodeGenerator(),
		analyzer:    analyzer,
		varOffsets:  make(map[string]int32),
		stackOffset: 0,
	}
}

// Generate generates machine code from AST
func (ng *NativeCodeGenerator) Generate(program *parser.Program) error {
	// Generate code for all functions
	for _, fn := range program.Functions {
		if err := ng.generateFunction(fn); err != nil {
			return err
		}
	}

	// Apply all fixups
	if err := ng.machine.ApplyFixups(); err != nil {
		return err
	}

	return nil
}

// generateFunction generates machine code for a function
func (ng *NativeCodeGenerator) generateFunction(fn *parser.FunctionDecl) error {
	// Reset state for new function
	ng.varOffsets = make(map[string]int32)
	ng.stackOffset = 0

	// Emit function label
	if fn.Name == "main" {
		ng.machine.EmitLabel("_start") // For ELF
		ng.machine.EmitLabel("main")   // For PE
	} else {
		ng.machine.EmitLabel(fn.Name)
	}

	// Function prologue
	ng.machine.EmitPushR64(RegRBP)                    // push rbp
	ng.machine.EmitMovR64R64(RegRBP, RegRSP)          // mov rbp, rsp
	ng.machine.EmitSubR64Imm(RegRSP, 256)             // sub rsp, 256 (reserve stack space)

	// Process function body
	if err := ng.generateBlock(fn.Body); err != nil {
		return err
	}

	// Function epilogue (fallthrough return)
	ng.machine.EmitMovR64R64(RegRSP, RegRBP)          // mov rsp, rbp
	ng.machine.EmitPopR64(RegRBP)                     // pop rbp
	ng.machine.EmitRet()                              // ret

	return nil
}

// generateBlock generates code for a block statement
func (ng *NativeCodeGenerator) generateBlock(block *parser.BlockStmt) error {
	for _, stmt := range block.Statements {
		if err := ng.generateStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

// generateStatement generates code for a statement
func (ng *NativeCodeGenerator) generateStatement(stmt parser.Statement) error {
	switch s := stmt.(type) {
	case *parser.VarDeclStmt:
		return ng.generateVarDecl(s)
	case *parser.AssignStmt:
		return ng.generateAssignment(s)
	case *parser.ReturnStmt:
		return ng.generateReturn(s)
	case *parser.IfStmt:
		return ng.generateIf(s)
	case *parser.WhileStmt:
		return ng.generateWhile(s)
	case *parser.ExprStmt:
		return ng.generateExprStmt(s)
	case *parser.BlockStmt:
		return ng.generateBlock(s)
	default:
		return fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

// generateVarDecl generates code for variable declaration
func (ng *NativeCodeGenerator) generateVarDecl(decl *parser.VarDeclStmt) error {
	// Allocate stack space
	ng.stackOffset -= 8
	ng.varOffsets[decl.Name] = ng.stackOffset

	// Generate initializer if present
	if decl.Value != nil {
		var varType types.Type
		if decl.Type != "" {
			varType = types.FromString(decl.Type)
		} else {
			varType = ng.analyzer.GetExprType(decl.Value)
		}

		if err := ng.generateExpression(decl.Value); err != nil {
			return err
		}

		// Store to variable (result is in EAX for int)
		if !varType.Equals(types.FloatType) {
			ng.machine.EmitMovMemR32(RegRBP, ng.stackOffset, RegRAX&0x7)
		}
	}

	return nil
}

// generateAssignment generates code for assignment
func (ng *NativeCodeGenerator) generateAssignment(assign *parser.AssignStmt) error {
	varType := ng.analyzer.GetExprType(assign.Value)
	if varType == nil {
		return fmt.Errorf("cannot determine type for variable: %s", assign.Name)
	}

	// Generate right-hand side
	if err := ng.generateExpression(assign.Value); err != nil {
		return err
	}

	// Allocate stack space if first assignment
	offset, ok := ng.varOffsets[assign.Name]
	if !ok {
		ng.stackOffset -= 8
		offset = ng.stackOffset
		ng.varOffsets[assign.Name] = offset
	}

	// Store result (EAX) to variable
	if !varType.Equals(types.FloatType) {
		ng.machine.EmitMovMemR32(RegRBP, offset, RegRAX&0x7)
	}

	return nil
}

// generateReturn generates code for return statement
func (ng *NativeCodeGenerator) generateReturn(ret *parser.ReturnStmt) error {
	if ret.Value != nil {
		if err := ng.generateExpression(ret.Value); err != nil {
			return err
		}
	}

	// Epilogue
	ng.machine.EmitMovR64R64(RegRSP, RegRBP)
	ng.machine.EmitPopR64(RegRBP)
	ng.machine.EmitRet()

	return nil
}

// generateIf generates code for if statement
func (ng *NativeCodeGenerator) generateIf(ifStmt *parser.IfStmt) error {
	elseLabel := ng.newLabel("else")
	endLabel := ng.newLabel("endif")

	// Generate condition
	if err := ng.generateExpression(ifStmt.Condition); err != nil {
		return err
	}

	// Test and jump
	ng.machine.EmitTestR32R32(RegRAX&0x7, RegRAX&0x7)
	if ifStmt.ElseBranch != nil {
		ng.machine.EmitJe(elseLabel)
	} else {
		ng.machine.EmitJe(endLabel)
	}

	// Then branch
	if err := ng.generateBlock(ifStmt.ThenBranch); err != nil {
		return err
	}

	if ifStmt.ElseBranch != nil {
		ng.machine.EmitJmp(endLabel)
		ng.machine.EmitLabel(elseLabel)
		if err := ng.generateBlock(ifStmt.ElseBranch); err != nil {
			return err
		}
	}

	ng.machine.EmitLabel(endLabel)
	return nil
}

// generateWhile generates code for while loop
func (ng *NativeCodeGenerator) generateWhile(whileStmt *parser.WhileStmt) error {
	condLabel := ng.newLabel("while_cond")
	endLabel := ng.newLabel("while_end")

	ng.machine.EmitLabel(condLabel)

	// Generate condition
	if err := ng.generateExpression(whileStmt.Condition); err != nil {
		return err
	}

	// Test and jump
	ng.machine.EmitTestR32R32(RegRAX&0x7, RegRAX&0x7)
	ng.machine.EmitJe(endLabel)

	// Body
	if err := ng.generateBlock(whileStmt.Body); err != nil {
		return err
	}

	ng.machine.EmitJmp(condLabel)
	ng.machine.EmitLabel(endLabel)

	return nil
}

// generateExprStmt generates code for expression statement
func (ng *NativeCodeGenerator) generateExprStmt(stmt *parser.ExprStmt) error {
	return ng.generateExpression(stmt.Expression)
}

// generateExpression generates code for an expression (result in EAX)
func (ng *NativeCodeGenerator) generateExpression(expr parser.Expression) error {
	switch e := expr.(type) {
	case *parser.IntLiteral:
		// mov eax, <value>
		ng.machine.EmitMovR32Imm32(RegRAX&0x7, int32(e.Value))
		return nil

	case *parser.Identifier:
		// Load from variable: mov eax, [rbp+offset]
		offset, ok := ng.varOffsets[e.Name]
		if !ok {
			return fmt.Errorf("undefined variable: %s", e.Name)
		}
		ng.machine.EmitMovR32Mem(RegRAX&0x7, RegRBP, offset)
		return nil

	case *parser.BinaryExpr:
		return ng.generateBinaryExpr(e)

	case *parser.UnaryExpr:
		return ng.generateUnaryExpr(e)

	default:
		return fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// generateBinaryExpr generates code for binary expression
func (ng *NativeCodeGenerator) generateBinaryExpr(expr *parser.BinaryExpr) error {
	// Generate left operand
	if err := ng.generateExpression(expr.Left); err != nil {
		return err
	}

	// Push left result
	ng.machine.EmitPushR64(RegRAX)

	// Generate right operand
	if err := ng.generateExpression(expr.Right); err != nil {
		return err
	}

	// Pop left operand into RBX
	ng.machine.EmitPopR64(RegRBX)

	// Perform operation (RBX = left, RAX = right)
	switch expr.Operator {
	case "+":
		ng.machine.EmitAddR32R32(RegRAX&0x7, RegRBX&0x7) // eax = eax + ebx
	case "-":
		// eax = ebx - eax
		ng.machine.EmitSubR32R32(RegRBX&0x7, RegRAX&0x7) // ebx = ebx - eax
		ng.machine.EmitMovR32R32(RegRAX&0x7, RegRBX&0x7)  // eax = ebx
	default:
		return fmt.Errorf("unsupported binary operator: %s", expr.Operator)
	}

	return nil
}

// generateUnaryExpr generates code for unary expression
func (ng *NativeCodeGenerator) generateUnaryExpr(expr *parser.UnaryExpr) error {
	if err := ng.generateExpression(expr.Operand); err != nil {
		return err
	}

	switch expr.Operator {
	case "-":
		ng.machine.EmitNegR32(RegRAX & 0x7)
	default:
		return fmt.Errorf("unsupported unary operator: %s", expr.Operator)
	}

	return nil
}

// newLabel generates a unique label
func (ng *NativeCodeGenerator) newLabel(prefix string) string {
	ng.labelCounter++
	return fmt.Sprintf(".L%s_%d", prefix, ng.labelCounter)
}

// GetMachineCode returns the generated machine code
func (ng *NativeCodeGenerator) GetMachineCode() []byte {
	return ng.machine.GetCode()
}

// BuildExecutable builds an executable file from the generated machine code
func (ng *NativeCodeGenerator) BuildExecutable(outputPath string) error {
	code := ng.GetMachineCode()

	if runtime.GOOS == "windows" {
		// Generate PE file
		pe := NewPEGenerator(code)
		return pe.Generate(outputPath)
	} else {
		// Generate ELF file
		elf := NewELFGenerator(code)
		return elf.Generate(outputPath)
	}
}
