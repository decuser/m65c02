package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"m65c02/internal/machine"
)

// traceInstructions derives the instruction from recorded bus reads only.
// Incomplete operands at a ring boundary remain explicitly unknown.
func traceInstructions(rows []machine.BusCycle) []string {
	context := make([]string, len(rows))
	current := "(fetch outside capture)"
	for i, c := range rows {
		if c.Sync {
			operands := make([]string, 0)
			for n := 1; n < c.Length; n++ {
				value := "??"
				for j := i + 1; j < len(rows) && !rows[j].Sync; j++ {
					if !rows[j].Write && rows[j].Address == c.Address+uint16(n) {
						value = fmt.Sprintf("%02X", rows[j].Data)
						break
					}
				}
				operands = append(operands, value)
			}
			current = fmt.Sprintf("%04X: %02X", c.Address, c.Data)
			if len(operands) > 0 {
				current += " " + strings.Join(operands, " ")
			}
			current += "  " + c.Mnemonic
			if arg := traceOperand(c, operands); arg != "" {
				current += " " + arg
			}
		}
		context[i] = current
	}
	return context
}

func traceOperand(c machine.BusCycle, operands []string) string {
	if len(operands) == 0 {
		if c.Mode == "acc" {
			return "A"
		}
		return ""
	}
	zp := "$" + operands[0]
	word := "$????"
	if len(operands) > 1 {
		word = "$" + operands[1] + operands[0]
	}
	switch c.Mode {
	case "imm":
		return "#" + zp
	case "zp":
		return zp
	case "zpx":
		return zp + ",X"
	case "zpy":
		return zp + ",Y"
	case "abs":
		return word
	case "abx":
		return word + ",X"
	case "aby":
		return word + ",Y"
	case "ix":
		return "(" + zp + ",X)"
	case "iy":
		return "(" + zp + "),Y"
	case "iz":
		return "(" + zp + ")"
	case "ind":
		return "(" + word + ")"
	case "inx":
		return "(" + word + ",X)"
	case "rel":
		offset, err := strconv.ParseUint(operands[0], 16, 8)
		if err != nil {
			return "$????"
		}
		return fmt.Sprintf("$%04X", uint16(int32(c.Address)+2+int32(int8(offset))))
	}
	return ""
}

func pinFields(c machine.BusCycle) []string {
	if c.Pins == nil {
		return []string{"", "", "", "", "", "", "", "", "", ""}
	}
	p := c.Pins
	return []string{fmt.Sprintf("%02X", p.R002.PA), fmt.Sprintf("%02X", p.R002.PB),
		fmt.Sprintf("%02X", p.R002.DDRA), fmt.Sprintf("%02X", p.R002.DDRB),
		fmt.Sprintf("%02X", p.R003.PA), fmt.Sprintf("%02X", p.R003.PB),
		fmt.Sprintf("%02X", p.R003.DDRA), fmt.Sprintf("%02X", p.R003.DDRB),
		fmt.Sprint(p.Digit), fmt.Sprintf("%02X", p.Segments)}
}

func writeTraceCSV(w io.Writer, rows []machine.BusCycle) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"cycle", "address_hex", "data_hex", "rw", "sync", "irq", "nmi", "reset", "rdy", "instruction", "riot002_pa", "riot002_pb", "riot002_ddra", "riot002_ddrb", "riot003_pa", "riot003_pb", "riot003_ddra", "riot003_ddrb", "led_digit_zero_based", "led_segments_hex"}); err != nil {
		return err
	}
	context := traceInstructions(rows)
	for i, c := range rows {
		rw := "R"
		if c.Write {
			rw = "W"
		}
		line := []string{fmt.Sprint(c.Cycle), fmt.Sprintf("%04X", c.Address), fmt.Sprintf("%02X", c.Data), rw,
			fmt.Sprint(c.Sync), fmt.Sprint(c.IRQ), fmt.Sprint(c.NMI), fmt.Sprint(c.Reset), fmt.Sprint(c.Ready), context[i]}
		line = append(line, pinFields(c)...)
		if err := cw.Write(line); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func writeTraceText(w io.Writer, rows []machine.BusCycle) error {
	if _, err := fmt.Fprintf(w, "KIM-1 logic analyzer · %d retained cycles\n", len(rows)); err != nil {
		return err
	}
	if len(rows) > 0 {
		if _, err := fmt.Fprintf(w, "Cycles %d through %d (inclusive)\n", rows[0].Cycle, rows[len(rows)-1].Cycle); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "CPU cycle | address data R/W SYNC IRQ NMI RESET RDY | RIOT002 PA PB DDRA DDRB | RIOT003 PA PB DDRA DDRB | LED digit segments"); err != nil {
		return err
	}
	context := traceInstructions(rows)
	for i, c := range rows {
		if c.Sync {
			if _, err := fmt.Fprintf(w, "\n▶ %s\n", context[i]); err != nil {
				return err
			}
		}
		rw := 'R'
		if c.Write {
			rw = 'W'
		}
		p := pinFields(c)
		if _, err := fmt.Fprintf(w, "%10d | %04X    %02X   %c   %t    %t   %t   %t     %t | %s %s %s %s | %s %s %s %s | %s %s\n",
			c.Cycle, c.Address, c.Data, rw, c.Sync, c.IRQ, c.NMI, c.Reset, c.Ready,
			p[0], p[1], p[2], p[3], p[4], p[5], p[6], p[7], p[8], p[9]); err != nil {
			return err
		}
	}
	return nil
}

func (p *panelServer) exportTrace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}
	format := r.URL.Query().Get("format")
	if format != "csv" && format != "text" {
		http.Error(w, "format must be csv or text", http.StatusBadRequest)
		return
	}
	p.mu.Lock()
	p.running = false // atomically freeze the producer and its ring before copying
	var rows []machine.BusCycle
	if p.kim != nil && p.kim.Bus.Trace != nil {
		rows = p.kim.Bus.Trace.Window(0, 8192)
	}
	p.mu.Unlock()
	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=kim1-trace.csv")
		_ = writeTraceCSV(w, rows)
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=kim1-trace.txt")
		_ = writeTraceText(w, rows)
	}
}
