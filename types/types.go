package types

// Type represents a type in the language
type Type interface {
	String() string
	Equals(Type) bool
}

// BasicType represents a basic type (int, float)
type BasicType struct {
	Name string
}

func (bt *BasicType) String() string {
	return bt.Name
}

func (bt *BasicType) Equals(other Type) bool {
	if otherBasic, ok := other.(*BasicType); ok {
		return bt.Name == otherBasic.Name
	}
	return false
}

// Predefined types
var (
	IntType   = &BasicType{Name: "int"}
	FloatType = &BasicType{Name: "float"}
	VoidType  = &BasicType{Name: "void"}
)

// FromString converts a string to a Type
func FromString(s string) Type {
	switch s {
	case "int":
		return IntType
	case "float":
		return FloatType
	case "void":
		return VoidType
	default:
		return nil
	}
}

// InferFromLiteral infers the type from a literal token type
func InferFromLiteral(tokenType string) Type {
	switch tokenType {
	case "INT_LIT":
		return IntType
	case "FLOAT_LIT":
		return FloatType
	default:
		return nil
	}
}

// IsNumeric checks if a type is numeric (int or float)
func IsNumeric(t Type) bool {
	return t.Equals(IntType) || t.Equals(FloatType)
}

// CanAssign checks if a value of type 'from' can be assigned to type 'to'
func CanAssign(to, from Type) bool {
	// Strict type checking: types must match exactly
	return to.Equals(from)
}

// GetBinaryOpResultType determines the result type of a binary operation
func GetBinaryOpResultType(left, right Type, op string) (Type, error) {
	// For MVP, we require exact type matching
	if !left.Equals(right) {
		return nil, &TypeError{
			Message: "type mismatch in binary operation: " + left.String() + " and " + right.String(),
		}
	}

	// Comparison operators always return int (0 or 1, like C)
	if isComparisonOp(op) {
		return IntType, nil
	}

	// Arithmetic and logical operators return the operand type
	return left, nil
}

// GetUnaryOpResultType determines the result type of a unary operation
func GetUnaryOpResultType(operand Type, op string) (Type, error) {
	if !IsNumeric(operand) {
		return nil, &TypeError{
			Message: "unary operator " + op + " requires numeric type",
		}
	}
	return operand, nil
}

func isComparisonOp(op string) bool {
	return op == "==" || op == "!=" || op == "<" || op == "<=" || op == ">" || op == ">="
}

// TypeError represents a type error
type TypeError struct {
	Message string
}

func (te *TypeError) Error() string {
	return te.Message
}
