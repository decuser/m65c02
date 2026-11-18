package main

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"m65c02/internal/machine"
)

func TestTraceExportBothRRIOTs(t *testing.T) {
	rows := []machine.BusCycle{
		{Cycle: 7, Address: 0x1f50, Data: 0x8c, Sync: true, Mnemonic: "STY", Mode: "abs", Length: 3},
		{Cycle: 8, Address: 0x1f51, Data: 0x40},
		{Cycle: 9, Address: 0x1f52, Data: 0x17},
		{Cycle: 10, Address: 0x1740, Data: 0xf1, Write: true, Pins: &machine.BoardPins{
			R002:  machine.RRIOTPins{PA: 0xf1, PB: 0xd3, DDRA: 0x7f, DDRB: 0x3f},
			R003:  machine.RRIOTPins{PA: 0x12, PB: 0x34, DDRA: 0xff, DDRB: 0xee},
			Digit: 5, Segments: 0x71,
		}},
	}
	var b bytes.Buffer
	if err := writeTraceCSV(&b, rows); err != nil {
		t.Fatal(err)
	}
	csvRows, err := csv.NewReader(&b).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(csvRows) != 5 || len(csvRows[0]) != 20 || csvRows[4][0] != "10" || csvRows[4][10] != "F1" || csvRows[4][14] != "12" || csvRows[4][18] != "5" || csvRows[4][19] != "71" {
		t.Fatalf("unexpected CSV output: %v", csvRows)
	}
	if !strings.Contains(csvRows[4][9], "STY") {
		t.Fatalf("lost instruction context: %v", csvRows[4][9])
	}
	b.Reset()
	if err := writeTraceText(&b, rows); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "▶ 1F50: 8C 40 17  STY") || !strings.Contains(b.String(), "F1 D3 7F 3F | 12 34 FF EE | 5 71") {
		t.Fatalf("unexpected text output: %s", b.String())
	}
}

func TestTraceExportRejectsInvalidFormat(t *testing.T) {
	p := &panelServer{}
	w := httptest.NewRecorder()
	p.exportTrace(w, httptest.NewRequest(http.MethodGet, "/api/trace/export?format=zip", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("got status %d", w.Code)
	}
}
