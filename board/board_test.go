package board

import (
	"strings"
	"testing"
)

func TestParseBoardIDValid(t *testing.T) {
	want := InitialBoard()
	got, err := ParseBoardID(want.ID())
	if err != nil {
		t.Fatalf("ParseBoardID(%q) returned error: %v", want.ID(), err)
	}
	if got.Size != want.Size {
		t.Fatalf("Size = %d, want %d", got.Size, want.Size)
	}
	if got.Turn != want.Turn {
		t.Fatalf("Turn = %d, want %d", got.Turn, want.Turn)
	}
	if string(got.Cells) != string(want.Cells) {
		t.Fatalf("Cells = %v, want %v", got.Cells, want.Cells)
	}
	if got.Status != want.Status {
		t.Fatalf("Status = %q, want %q", got.Status, want.Status)
	}
}

func TestParseBoardIDAcceptsValidIDs(t *testing.T) {
	tests := []string{
		InitialBoard().ID(),
		"2-2-" + strings.Repeat("01", 16),
		"2-3-" + strings.Repeat("02", 16),
		"4-1-" + strings.Repeat("00", 64),
	}

	for _, id := range tests {
		if _, err := ParseBoardID(id); err != nil {
			t.Fatalf("ParseBoardID(%q) returned error: %v", id, err)
		}
	}
}

func TestParseBoardIDRejectsInvalidIDs(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{name: "not three parts", id: "invalid"},
		{name: "unsupported size", id: "1-1-0000"},
		{name: "invalid turn", id: "2-0-0000000000000000"},
		{name: "payload length mismatch", id: "2-1-00"},
		{name: "malformed hex", id: "2-1-zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"},
		{name: "cell value out of range", id: "2-1-" + strings.Repeat("03", 16)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseBoardID(tt.id); err == nil {
				t.Fatalf("ParseBoardID(%q) = nil error, want invalid board ID error", tt.id)
			}
		})
	}
}
