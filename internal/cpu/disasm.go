package cpu

// DecodeOpcode describes an opcode without reading memory or changing CPU state.
// Operand bytes are obtained from the subsequent captured bus transfers.
func DecodeOpcode(op byte) (mnemonic, mode string, length int) {
	names := [...]string{
		"ADC", "AND", "ASL", "BCC", "BCS", "BEQ", "BIT", "BMI", "BNE", "BPL", "BRA", "BRK",
		"BVC", "BVS", "CLC", "CLD", "CLI", "CLV", "CMP", "CPX", "CPY", "DEC", "DEX", "DEY",
		"EOR", "INC", "INX", "INY", "JMP", "JSR", "LDA", "LDX", "LDY", "LSR", "NOP", "ORA",
		"PHA", "PHP", "PHX", "PHY", "PLA", "PLP", "PLX", "PLY", "ROL", "ROR", "RTI", "RTS",
		"SBC", "SEC", "SED", "SEI", "STA", "STX", "STY", "STZ", "TAX", "TAY", "TRB", "TSB",
		"TSX", "TXA", "TXS", "TYA",
	}
	i := instructionTable[op]
	mnemonic = names[i.kind]
	switch i.mode {
	case modeOne:
		return mnemonic, "impl", 1
	case modeTwo:
		return mnemonic, "impl", 2
	case modeImm:
		return mnemonic, "imm", 2
	case modeAcc:
		return mnemonic, "acc", 1
	case modeZp:
		return mnemonic, "zp", 2
	case modeZpx:
		return mnemonic, "zpx", 2
	case modeZpy:
		return mnemonic, "zpy", 2
	case modeAbs:
		return mnemonic, "abs", 3
	case modeAbx:
		return mnemonic, "abx", 3
	case modeAby:
		return mnemonic, "aby", 3
	case modeIx:
		return mnemonic, "ix", 2
	case modeIy:
		return mnemonic, "iy", 2
	case modeIz:
		return mnemonic, "iz", 2
	case modeRel:
		return mnemonic, "rel", 2
	case modeAbsnop:
		return mnemonic, "abs", 3
	case modeAbsxno:
		return mnemonic, "abx", 3
	case modeInd:
		return mnemonic, "ind", 3
	case modeInx:
		return mnemonic, "inx", 3
	case modeImpl:
		if i.kind == insJSR {
			return mnemonic, "abs", 3
		}
		if i.kind == insBRK {
			return mnemonic, "impl", 2
		}
		return mnemonic, "impl", 1
	default:
		return mnemonic, "impl", 1
	}
}
