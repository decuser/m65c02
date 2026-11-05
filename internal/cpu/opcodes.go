// Opcode metadata for the Apple IIe-compatible Synertek CMOS profile.
// Cycle execution is performed by the shared CPU phase sequencer.
package cpu

type addrMode uint8

const (
	modeOne addrMode = iota
	modeTwo
	modeImm
	modeAcc
	modeZp
	modeZpx
	modeZpy
	modeAbs
	modeAbx
	modeAby
	modeIx
	modeIy
	modeIz
	modeRel
	modeAbsnop
	modeAbsxno
	modeInd
	modeInx
	modeImpl
)

type instrKind uint8

const (
	insADC instrKind = iota
	insAND
	insASL
	insBCC
	insBCS
	insBEQ
	insBIT
	insBMI
	insBNE
	insBPL
	insBRA
	insBRK
	insBVC
	insBVS
	insCLC
	insCLD
	insCLI
	insCLV
	insCMP
	insCPX
	insCPY
	insDEC
	insDEX
	insDEY
	insEOR
	insINC
	insINX
	insINY
	insJMP
	insJSR
	insLDA
	insLDX
	insLDY
	insLSR
	insNOP
	insORA
	insPHA
	insPHP
	insPHX
	insPHY
	insPLA
	insPLP
	insPLX
	insPLY
	insROL
	insROR
	insRTI
	insRTS
	insSBC
	insSEC
	insSED
	insSEI
	insSTA
	insSTX
	insSTY
	insSTZ
	insTAX
	insTAY
	insTRB
	insTSB
	insTSX
	insTXA
	insTXS
	insTYA
)

type instruction struct {
	kind instrKind
	mode addrMode
}

var instructionTable = [256]instruction{
	/* 00 */ {insBRK, modeImpl},
	/* 01 */ {insORA, modeIx},
	/* 02 */ {insNOP, modeImm},
	/* 03 */ {insNOP, modeOne},
	/* 04 */ {insTSB, modeZp},
	/* 05 */ {insORA, modeZp},
	/* 06 */ {insASL, modeZp},
	/* 07 */ {insNOP, modeZp},
	/* 08 */ {insPHP, modeImpl},
	/* 09 */ {insORA, modeImm},
	/* 0A */ {insASL, modeAcc},
	/* 0B */ {insNOP, modeOne},
	/* 0C */ {insTSB, modeAbs},
	/* 0D */ {insORA, modeAbs},
	/* 0E */ {insASL, modeAbs},
	/* 0F */ {insNOP, modeAbsnop},
	/* 10 */ {insBPL, modeRel},
	/* 11 */ {insORA, modeIy},
	/* 12 */ {insORA, modeIz},
	/* 13 */ {insNOP, modeOne},
	/* 14 */ {insTRB, modeZp},
	/* 15 */ {insORA, modeZpx},
	/* 16 */ {insASL, modeZpx},
	/* 17 */ {insNOP, modeZpx},
	/* 18 */ {insCLC, modeImpl},
	/* 19 */ {insORA, modeAby},
	/* 1A */ {insINC, modeAcc},
	/* 1B */ {insNOP, modeOne},
	/* 1C */ {insTRB, modeAbs},
	/* 1D */ {insORA, modeAbx},
	/* 1E */ {insASL, modeAbx},
	/* 1F */ {insNOP, modeAbsxno},
	/* 20 */ {insJSR, modeImpl},
	/* 21 */ {insAND, modeIx},
	/* 22 */ {insNOP, modeImm},
	/* 23 */ {insNOP, modeOne},
	/* 24 */ {insBIT, modeZp},
	/* 25 */ {insAND, modeZp},
	/* 26 */ {insROL, modeZp},
	/* 27 */ {insNOP, modeZp},
	/* 28 */ {insPLP, modeImpl},
	/* 29 */ {insAND, modeImm},
	/* 2A */ {insROL, modeAcc},
	/* 2B */ {insNOP, modeOne},
	/* 2C */ {insBIT, modeAbs},
	/* 2D */ {insAND, modeAbs},
	/* 2E */ {insROL, modeAbs},
	/* 2F */ {insNOP, modeAbsnop},
	/* 30 */ {insBMI, modeRel},
	/* 31 */ {insAND, modeIy},
	/* 32 */ {insAND, modeIz},
	/* 33 */ {insNOP, modeOne},
	/* 34 */ {insBIT, modeZpx},
	/* 35 */ {insAND, modeZpx},
	/* 36 */ {insROL, modeZpx},
	/* 37 */ {insNOP, modeZpx},
	/* 38 */ {insSEC, modeImpl},
	/* 39 */ {insAND, modeAby},
	/* 3A */ {insDEC, modeAcc},
	/* 3B */ {insNOP, modeOne},
	/* 3C */ {insBIT, modeAbx},
	/* 3D */ {insAND, modeAbx},
	/* 3E */ {insROL, modeAbx},
	/* 3F */ {insNOP, modeAbsxno},
	/* 40 */ {insRTI, modeImpl},
	/* 41 */ {insEOR, modeIx},
	/* 42 */ {insNOP, modeImm},
	/* 43 */ {insNOP, modeOne},
	/* 44 */ {insNOP, modeZp},
	/* 45 */ {insEOR, modeZp},
	/* 46 */ {insLSR, modeZp},
	/* 47 */ {insNOP, modeZp},
	/* 48 */ {insPHA, modeImpl},
	/* 49 */ {insEOR, modeImm},
	/* 4A */ {insLSR, modeAcc},
	/* 4B */ {insNOP, modeOne},
	/* 4C */ {insJMP, modeAbs},
	/* 4D */ {insEOR, modeAbs},
	/* 4E */ {insLSR, modeAbs},
	/* 4F */ {insNOP, modeAbsnop},
	/* 50 */ {insBVC, modeRel},
	/* 51 */ {insEOR, modeIy},
	/* 52 */ {insEOR, modeIz},
	/* 53 */ {insNOP, modeOne},
	/* 54 */ {insNOP, modeZpx},
	/* 55 */ {insEOR, modeZpx},
	/* 56 */ {insLSR, modeZpx},
	/* 57 */ {insNOP, modeZpx},
	/* 58 */ {insCLI, modeImpl},
	/* 59 */ {insEOR, modeAby},
	/* 5A */ {insPHY, modeImpl},
	/* 5B */ {insNOP, modeOne},
	/* 5C */ {insNOP, modeAbsxno},
	/* 5D */ {insEOR, modeAbx},
	/* 5E */ {insLSR, modeAbx},
	/* 5F */ {insNOP, modeAbsxno},
	/* 60 */ {insRTS, modeImpl},
	/* 61 */ {insADC, modeIx},
	/* 62 */ {insNOP, modeImm},
	/* 63 */ {insNOP, modeOne},
	/* 64 */ {insSTZ, modeZp},
	/* 65 */ {insADC, modeZp},
	/* 66 */ {insROR, modeZp},
	/* 67 */ {insNOP, modeZp},
	/* 68 */ {insPLA, modeImpl},
	/* 69 */ {insADC, modeImm},
	/* 6A */ {insROR, modeAcc},
	/* 6B */ {insNOP, modeOne},
	/* 6C */ {insJMP, modeInd},
	/* 6D */ {insADC, modeAbs},
	/* 6E */ {insROR, modeAbs},
	/* 6F */ {insNOP, modeAbsnop},
	/* 70 */ {insBVS, modeRel},
	/* 71 */ {insADC, modeIy},
	/* 72 */ {insADC, modeIz},
	/* 73 */ {insNOP, modeOne},
	/* 74 */ {insSTZ, modeZpx},
	/* 75 */ {insADC, modeZpx},
	/* 76 */ {insROR, modeZpx},
	/* 77 */ {insNOP, modeZpx},
	/* 78 */ {insSEI, modeImpl},
	/* 79 */ {insADC, modeAby},
	/* 7A */ {insPLY, modeImpl},
	/* 7B */ {insNOP, modeOne},
	/* 7C */ {insJMP, modeInx},
	/* 7D */ {insADC, modeAbx},
	/* 7E */ {insROR, modeAbx},
	/* 7F */ {insNOP, modeAbsxno},
	/* 80 */ {insBRA, modeRel},
	/* 81 */ {insSTA, modeIx},
	/* 82 */ {insNOP, modeImm},
	/* 83 */ {insNOP, modeOne},
	/* 84 */ {insSTY, modeZp},
	/* 85 */ {insSTA, modeZp},
	/* 86 */ {insSTX, modeZp},
	/* 87 */ {insNOP, modeZp},
	/* 88 */ {insDEY, modeImpl},
	/* 89 */ {insBIT, modeImm},
	/* 8A */ {insTXA, modeImpl},
	/* 8B */ {insNOP, modeOne},
	/* 8C */ {insSTY, modeAbs},
	/* 8D */ {insSTA, modeAbs},
	/* 8E */ {insSTX, modeAbs},
	/* 8F */ {insNOP, modeAbsnop},
	/* 90 */ {insBCC, modeRel},
	/* 91 */ {insSTA, modeIy},
	/* 92 */ {insSTA, modeIz},
	/* 93 */ {insNOP, modeOne},
	/* 94 */ {insSTY, modeZpx},
	/* 95 */ {insSTA, modeZpx},
	/* 96 */ {insSTX, modeZpy},
	/* 97 */ {insNOP, modeZpx},
	/* 98 */ {insTYA, modeImpl},
	/* 99 */ {insSTA, modeAby},
	/* 9A */ {insTXS, modeImpl},
	/* 9B */ {insNOP, modeOne},
	/* 9C */ {insSTZ, modeAbs},
	/* 9D */ {insSTA, modeAbx},
	/* 9E */ {insSTZ, modeAbx},
	/* 9F */ {insNOP, modeAbsxno},
	/* A0 */ {insLDY, modeImm},
	/* A1 */ {insLDA, modeIx},
	/* A2 */ {insLDX, modeImm},
	/* A3 */ {insNOP, modeOne},
	/* A4 */ {insLDY, modeZp},
	/* A5 */ {insLDA, modeZp},
	/* A6 */ {insLDX, modeZp},
	/* A7 */ {insNOP, modeZp},
	/* A8 */ {insTAY, modeImpl},
	/* A9 */ {insLDA, modeImm},
	/* AA */ {insTAX, modeImpl},
	/* AB */ {insNOP, modeOne},
	/* AC */ {insLDY, modeAbs},
	/* AD */ {insLDA, modeAbs},
	/* AE */ {insLDX, modeAbs},
	/* AF */ {insNOP, modeAbsnop},
	/* B0 */ {insBCS, modeRel},
	/* B1 */ {insLDA, modeIy},
	/* B2 */ {insLDA, modeIz},
	/* B3 */ {insNOP, modeOne},
	/* B4 */ {insLDY, modeZpx},
	/* B5 */ {insLDA, modeZpx},
	/* B6 */ {insLDX, modeZpy},
	/* B7 */ {insNOP, modeZpx},
	/* B8 */ {insCLV, modeImpl},
	/* B9 */ {insLDA, modeAby},
	/* BA */ {insTSX, modeImpl},
	/* BB */ {insNOP, modeOne},
	/* BC */ {insLDY, modeAbx},
	/* BD */ {insLDA, modeAbx},
	/* BE */ {insLDX, modeAby},
	/* BF */ {insNOP, modeAbsxno},
	/* C0 */ {insCPY, modeImm},
	/* C1 */ {insCMP, modeIx},
	/* C2 */ {insNOP, modeImm},
	/* C3 */ {insNOP, modeOne},
	/* C4 */ {insCPY, modeZp},
	/* C5 */ {insCMP, modeZp},
	/* C6 */ {insDEC, modeZp},
	/* C7 */ {insNOP, modeZp},
	/* C8 */ {insINY, modeImpl},
	/* C9 */ {insCMP, modeImm},
	/* CA */ {insDEX, modeImpl},
	/* CB */ {insNOP, modeImpl},
	/* CC */ {insCPY, modeAbs},
	/* CD */ {insCMP, modeAbs},
	/* CE */ {insDEC, modeAbs},
	/* CF */ {insNOP, modeAbsnop},
	/* D0 */ {insBNE, modeRel},
	/* D1 */ {insCMP, modeIy},
	/* D2 */ {insCMP, modeIz},
	/* D3 */ {insNOP, modeOne},
	/* D4 */ {insNOP, modeZpx},
	/* D5 */ {insCMP, modeZpx},
	/* D6 */ {insDEC, modeZpx},
	/* D7 */ {insNOP, modeZpx},
	/* D8 */ {insCLD, modeImpl},
	/* D9 */ {insCMP, modeAby},
	/* DA */ {insPHX, modeImpl},
	/* DB */ {insNOP, modeZpx},
	/* DC */ {insNOP, modeAbsxno},
	/* DD */ {insCMP, modeAbx},
	/* DE */ {insDEC, modeAbx},
	/* DF */ {insNOP, modeAbsxno},
	/* E0 */ {insCPX, modeImm},
	/* E1 */ {insSBC, modeIx},
	/* E2 */ {insNOP, modeImm},
	/* E3 */ {insNOP, modeOne},
	/* E4 */ {insCPX, modeZp},
	/* E5 */ {insSBC, modeZp},
	/* E6 */ {insINC, modeZp},
	/* E7 */ {insNOP, modeZp},
	/* E8 */ {insINX, modeImpl},
	/* E9 */ {insSBC, modeImm},
	/* EA */ {insNOP, modeTwo},
	/* EB */ {insNOP, modeOne},
	/* EC */ {insCPX, modeAbs},
	/* ED */ {insSBC, modeAbs},
	/* EE */ {insINC, modeAbs},
	/* EF */ {insNOP, modeAbsnop},
	/* F0 */ {insBEQ, modeRel},
	/* F1 */ {insSBC, modeIy},
	/* F2 */ {insSBC, modeIz},
	/* F3 */ {insNOP, modeOne},
	/* F4 */ {insNOP, modeZpx},
	/* F5 */ {insSBC, modeZpx},
	/* F6 */ {insINC, modeZpx},
	/* F7 */ {insNOP, modeZpx},
	/* F8 */ {insSED, modeImpl},
	/* F9 */ {insSBC, modeAby},
	/* FA */ {insPLX, modeImpl},
	/* FB */ {insNOP, modeOne},
	/* FC */ {insNOP, modeAbsxno},
	/* FD */ {insSBC, modeAbx},
	/* FE */ {insINC, modeAbx},
	/* FF */ {insNOP, modeAbsxno},
}
