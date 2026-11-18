package cpu

// MicroName describes the actual executed table operation in a trace event.
// Operation zero is the no-op/retire marker; bus resolution is a separate event.
func MicroName(operation uint8) string {
	names := [...]string{
		"NONE", "AD←PC", "AD←EA", "AD←STACK", "AD←VECTOR_LO", "AD←VECTOR_HI",
		"AD←BRANCH_DUMMY", "AD←PC−1", "DL←A", "DL←IR", "DL←LO", "DL←HI",
		"DL←OPERAND", "DL←OFFSET", "INC_PC", "DEC_S", "SET_NZ_A", "SET_NZ_OPERAND",
		"CALC_INDEXED", "INC_OPERAND", "CALC_BRANCH", "BRANCH_IF_SAME", "BRANCH_PC",
		"PC←HI:LO", "SET_I_CLR_D", "DO←PCH", "DO←PCL", "DO←P", "DO←A",
		"DO←OPERAND", "EXT_PREPARE", "EXT_COMMIT", "CONTROL_PREPARE", "CONTROL_COMMIT",
	}
	if int(operation) >= len(names) {
		return "UNKNOWN"
	}
	return names[operation]
}
