package codegen

import (
	"fmt"
	"goc/parser"
	"goc/semantic"
	"goc/types"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	llvmTypes "github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// Generator generates LLVM IR from an AST
type Generator struct {
	module      *ir.Module
	builder     *ir.Block
	currentFunc *ir.Func
	analyzer    *semantic.Analyzer
	variables   map[string]value.Value // Variable storage (alloca instructions)
	functions   map[string]*ir.Func
}

// New creates a new Generator
func New(analyzer *semantic.Analyzer) *Generator {
	return &Generator{
		module:    ir.NewModule(),
		analyzer:  analyzer,
		variables: make(map[string]value.Value),
		functions: make(map[string]*ir.Func),
	}
}

// Generate generates LLVM IR for the program
func (g *Generator) Generate(program *parser.Program) (*ir.Module, error) {
	// First pass: declare all functions
	for _, fn := range program.Functions {
		if err := g.declareFunctionSignature(fn); err != nil {
			return nil, err
		}
	}

	// Second pass: generate function bodies
	for _, fn := range program.Functions {
		if err := g.generateFunction(fn); err != nil {
			return nil, err
		}
	}

	return g.module, nil
}

// declareFunctionSignature declares a function signature
func (g *Generator) declareFunctionSignature(fn *parser.FunctionDecl) error {
	// Get return type
	returnType, err := g.getLLVMType(fn.ReturnType)
	if err != nil {
		return err
	}

	// Get parameter types
	var params []*ir.Param
	for _, param := range fn.Parameters {
		paramType, err := g.getLLVMType(param.Type)
		if err != nil {
			return err
		}
		params = append(params, ir.NewParam(param.Name, paramType))
	}

	// Create function
	llvmFunc := g.module.NewFunc(fn.Name, returnType, params...)
	g.functions[fn.Name] = llvmFunc

	return nil
}

// generateFunction generates LLVM IR for a function
func (g *Generator) generateFunction(fn *parser.FunctionDecl) error {
	llvmFunc := g.functions[fn.Name]
	g.currentFunc = llvmFunc

	// Create entry block
	entry := llvmFunc.NewBlock("entry")
	g.builder = entry

	// Clear variable map for new function
	g.variables = make(map[string]value.Value)

	// Allocate storage for parameters
	for i, param := range fn.Parameters {
		paramType, _ := g.getLLVMType(param.Type)
		alloca := entry.NewAlloca(paramType)
		alloca.LocalIdent = ir.NewLocalIdent(param.Name)
		entry.NewStore(llvmFunc.Params[i], alloca)
		g.variables[param.Name] = alloca
	}

	// Generate function body
	if err := g.generateBlockStmt(fn.Body); err != nil {
		return err
	}

	// Add default return if missing
	if g.builder.Term == nil {
		if fn.ReturnType == "int" {
			g.builder.NewRet(constant.NewInt(llvmTypes.I32, 0))
		} else if fn.ReturnType == "float" {
			g.builder.NewRet(constant.NewFloat(llvmTypes.Float, 0.0))
		}
	}

	return nil
}

// generateBlockStmt generates LLVM IR for a block statement
func (g *Generator) generateBlockStmt(block *parser.BlockStmt) error {
	for _, stmt := range block.Statements {
		if err := g.generateStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

// generateStatement generates LLVM IR for a statement
func (g *Generator) generateStatement(stmt parser.Statement) error {
	switch s := stmt.(type) {
	case *parser.VarDeclStmt:
		return g.generateVarDeclStmt(s)
	case *parser.AssignStmt:
		return g.generateAssignStmt(s)
	case *parser.ReturnStmt:
		return g.generateReturnStmt(s)
	case *parser.IfStmt:
		return g.generateIfStmt(s)
	case *parser.WhileStmt:
		return g.generateWhileStmt(s)
	case *parser.ExprStmt:
		_, err := g.generateExpression(s.Expression)
		return err
	case *parser.BlockStmt:
		return g.generateBlockStmt(s)
	default:
		return fmt.Errorf("unknown statement type")
	}
}

// generateVarDeclStmt generates LLVM IR for a variable declaration
func (g *Generator) generateVarDeclStmt(stmt *parser.VarDeclStmt) error {
	// Get LLVM type
	llvmType, err := g.getLLVMType(stmt.Type)
	if err != nil {
		return err
	}

	// Allocate storage
	alloca := g.builder.NewAlloca(llvmType)
	alloca.LocalIdent = ir.NewLocalIdent(stmt.Name)
	g.variables[stmt.Name] = alloca

	// Initialize if value provided
	if stmt.Value != nil {
		value, err := g.generateExpression(stmt.Value)
		if err != nil {
			return err
		}
		g.builder.NewStore(value, alloca)
	}

	return nil
}

// generateAssignStmt generates LLVM IR for an assignment statement
func (g *Generator) generateAssignStmt(stmt *parser.AssignStmt) error {
	// Check if variable exists (for type inference)
	alloca, exists := g.variables[stmt.Name]

	// Generate value
	val, err := g.generateExpression(stmt.Value)
	if err != nil {
		return err
	}

	if !exists {
		// Type-inferred declaration
		exprType := g.analyzer.GetExprType(stmt.Value)
		llvmType, err := g.typeToLLVMType(exprType)
		if err != nil {
			return err
		}
		newAlloca := g.builder.NewAlloca(llvmType)
		newAlloca.LocalIdent = ir.NewLocalIdent(stmt.Name)
		alloca = newAlloca
		g.variables[stmt.Name] = alloca
	}

	g.builder.NewStore(val, alloca)
	return nil
}

// generateReturnStmt generates LLVM IR for a return statement
func (g *Generator) generateReturnStmt(stmt *parser.ReturnStmt) error {
	if stmt.Value != nil {
		val, err := g.generateExpression(stmt.Value)
		if err != nil {
			return err
		}
		g.builder.NewRet(val)
	} else {
		g.builder.NewRet(nil)
	}
	return nil
}

// generateIfStmt generates LLVM IR for an if statement
func (g *Generator) generateIfStmt(stmt *parser.IfStmt) error {
	// Generate condition
	cond, err := g.generateExpression(stmt.Condition)
	if err != nil {
		return err
	}

	// Convert condition to boolean (non-zero = true)
	condType := g.analyzer.GetExprType(stmt.Condition)
	var boolCond value.Value
	if condType.Equals(types.IntType) {
		zero := constant.NewInt(llvmTypes.I32, 0)
		boolCond = g.builder.NewICmp(enum.IPredNE, cond, zero)
	} else if condType.Equals(types.FloatType) {
		zero := constant.NewFloat(llvmTypes.Float, 0.0)
		boolCond = g.builder.NewFCmp(enum.FPredONE, cond, zero)
	}

	// Create basic blocks
	thenBlock := g.currentFunc.NewBlock("then")
	mergeBlock := g.currentFunc.NewBlock("merge")
	var elseBlock *ir.Block
	if stmt.ElseBranch != nil {
		elseBlock = g.currentFunc.NewBlock("else")
		g.builder.NewCondBr(boolCond, thenBlock, elseBlock)
	} else {
		g.builder.NewCondBr(boolCond, thenBlock, mergeBlock)
	}

	// Generate then branch
	g.builder = thenBlock
	if err := g.generateBlockStmt(stmt.ThenBranch); err != nil {
		return err
	}
	if g.builder.Term == nil {
		g.builder.NewBr(mergeBlock)
	}

	// Generate else branch if exists
	if stmt.ElseBranch != nil {
		g.builder = elseBlock
		if err := g.generateBlockStmt(stmt.ElseBranch); err != nil {
			return err
		}
		if g.builder.Term == nil {
			g.builder.NewBr(mergeBlock)
		}
	}

	// Continue with merge block
	g.builder = mergeBlock
	return nil
}

// generateWhileStmt generates LLVM IR for a while loop
func (g *Generator) generateWhileStmt(stmt *parser.WhileStmt) error {
	// Create basic blocks
	condBlock := g.currentFunc.NewBlock("while.cond")
	bodyBlock := g.currentFunc.NewBlock("while.body")
	endBlock := g.currentFunc.NewBlock("while.end")

	// Branch to condition
	g.builder.NewBr(condBlock)

	// Generate condition
	g.builder = condBlock
	cond, err := g.generateExpression(stmt.Condition)
	if err != nil {
		return err
	}

	// Convert condition to boolean
	condType := g.analyzer.GetExprType(stmt.Condition)
	var boolCond value.Value
	if condType.Equals(types.IntType) {
		zero := constant.NewInt(llvmTypes.I32, 0)
		boolCond = g.builder.NewICmp(enum.IPredNE, cond, zero)
	} else if condType.Equals(types.FloatType) {
		zero := constant.NewFloat(llvmTypes.Float, 0.0)
		boolCond = g.builder.NewFCmp(enum.FPredONE, cond, zero)
	}

	g.builder.NewCondBr(boolCond, bodyBlock, endBlock)

	// Generate body
	g.builder = bodyBlock
	if err := g.generateBlockStmt(stmt.Body); err != nil {
		return err
	}
	if g.builder.Term == nil {
		g.builder.NewBr(condBlock)
	}

	// Continue with end block
	g.builder = endBlock
	return nil
}

// generateExpression generates LLVM IR for an expression
func (g *Generator) generateExpression(expr parser.Expression) (value.Value, error) {
	switch e := expr.(type) {
	case *parser.IntLiteral:
		return constant.NewInt(llvmTypes.I32, e.Value), nil
	case *parser.FloatLiteral:
		return constant.NewFloat(llvmTypes.Float, e.Value), nil
	case *parser.Identifier:
		return g.generateIdentifier(e)
	case *parser.BinaryExpr:
		return g.generateBinaryExpr(e)
	case *parser.UnaryExpr:
		return g.generateUnaryExpr(e)
	case *parser.CallExpr:
		return g.generateCallExpr(e)
	default:
		return nil, fmt.Errorf("unknown expression type")
	}
}

// generateIdentifier generates LLVM IR for an identifier
func (g *Generator) generateIdentifier(ident *parser.Identifier) (value.Value, error) {
	alloca, exists := g.variables[ident.Name]
	if !exists {
		return nil, fmt.Errorf("undefined variable: %s", ident.Name)
	}
	return g.builder.NewLoad(alloca.Type().(*llvmTypes.PointerType).ElemType, alloca), nil
}

// generateBinaryExpr generates LLVM IR for a binary expression
func (g *Generator) generateBinaryExpr(expr *parser.BinaryExpr) (value.Value, error) {
	left, err := g.generateExpression(expr.Left)
	if err != nil {
		return nil, err
	}

	right, err := g.generateExpression(expr.Right)
	if err != nil {
		return nil, err
	}

	exprType := g.analyzer.GetExprType(expr.Left)

	if exprType.Equals(types.IntType) {
		return g.generateIntBinaryOp(left, right, expr.Operator)
	} else if exprType.Equals(types.FloatType) {
		return g.generateFloatBinaryOp(left, right, expr.Operator)
	}

	return nil, fmt.Errorf("unsupported binary operation type")
}

// generateIntBinaryOp generates LLVM IR for integer binary operations
func (g *Generator) generateIntBinaryOp(left, right value.Value, op string) (value.Value, error) {
	switch op {
	case "+":
		return g.builder.NewAdd(left, right), nil
	case "-":
		return g.builder.NewSub(left, right), nil
	case "*":
		return g.builder.NewMul(left, right), nil
	case "/":
		return g.builder.NewSDiv(left, right), nil
	case "%":
		return g.builder.NewSRem(left, right), nil
	case "==":
		return g.builder.NewICmp(enum.IPredEQ, left, right), nil
	case "!=":
		return g.builder.NewICmp(enum.IPredNE, left, right), nil
	case "<":
		return g.builder.NewICmp(enum.IPredSLT, left, right), nil
	case "<=":
		return g.builder.NewICmp(enum.IPredSLE, left, right), nil
	case ">":
		return g.builder.NewICmp(enum.IPredSGT, left, right), nil
	case ">=":
		return g.builder.NewICmp(enum.IPredSGE, left, right), nil
	case "&&":
		return g.builder.NewAnd(left, right), nil
	case "||":
		return g.builder.NewOr(left, right), nil
	default:
		return nil, fmt.Errorf("unknown integer binary operator: %s", op)
	}
}

// generateFloatBinaryOp generates LLVM IR for float binary operations
func (g *Generator) generateFloatBinaryOp(left, right value.Value, op string) (value.Value, error) {
	switch op {
	case "+":
		return g.builder.NewFAdd(left, right), nil
	case "-":
		return g.builder.NewFSub(left, right), nil
	case "*":
		return g.builder.NewFMul(left, right), nil
	case "/":
		return g.builder.NewFDiv(left, right), nil
	case "==":
		return g.builder.NewFCmp(enum.FPredOEQ, left, right), nil
	case "!=":
		return g.builder.NewFCmp(enum.FPredONE, left, right), nil
	case "<":
		return g.builder.NewFCmp(enum.FPredOLT, left, right), nil
	case "<=":
		return g.builder.NewFCmp(enum.FPredOLE, left, right), nil
	case ">":
		return g.builder.NewFCmp(enum.FPredOGT, left, right), nil
	case ">=":
		return g.builder.NewFCmp(enum.FPredOGE, left, right), nil
	default:
		return nil, fmt.Errorf("unsupported float binary operator: %s", op)
	}
}

// generateUnaryExpr generates LLVM IR for a unary expression
func (g *Generator) generateUnaryExpr(expr *parser.UnaryExpr) (value.Value, error) {
	operand, err := g.generateExpression(expr.Operand)
	if err != nil {
		return nil, err
	}

	exprType := g.analyzer.GetExprType(expr.Operand)

	switch expr.Operator {
	case "-":
		if exprType.Equals(types.IntType) {
			zero := constant.NewInt(llvmTypes.I32, 0)
			return g.builder.NewSub(zero, operand), nil
		} else if exprType.Equals(types.FloatType) {
			zero := constant.NewFloat(llvmTypes.Float, 0.0)
			return g.builder.NewFSub(zero, operand), nil
		}
	case "!":
		if exprType.Equals(types.IntType) {
			zero := constant.NewInt(llvmTypes.I32, 0)
			return g.builder.NewICmp(enum.IPredEQ, operand, zero), nil
		}
	}

	return nil, fmt.Errorf("unsupported unary operator: %s", expr.Operator)
}

// generateCallExpr generates LLVM IR for a function call
func (g *Generator) generateCallExpr(expr *parser.CallExpr) (value.Value, error) {
	fn, exists := g.functions[expr.Function]
	if !exists {
		return nil, fmt.Errorf("undefined function: %s", expr.Function)
	}

	var args []value.Value
	for _, arg := range expr.Arguments {
		val, err := g.generateExpression(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, val)
	}

	return g.builder.NewCall(fn, args...), nil
}

// getLLVMType converts a type string to an LLVM type
func (g *Generator) getLLVMType(typeStr string) (llvmTypes.Type, error) {
	switch typeStr {
	case "int":
		return llvmTypes.I32, nil
	case "float":
		return llvmTypes.Float, nil
	case "void":
		return llvmTypes.Void, nil
	default:
		return nil, fmt.Errorf("unknown type: %s", typeStr)
	}
}

// typeToLLVMType converts a Type to an LLVM type
func (g *Generator) typeToLLVMType(t types.Type) (llvmTypes.Type, error) {
	if t.Equals(types.IntType) {
		return llvmTypes.I32, nil
	} else if t.Equals(types.FloatType) {
		return llvmTypes.Float, nil
	}
	return nil, fmt.Errorf("unknown type: %s", t.String())
}
