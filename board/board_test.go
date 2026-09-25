package board

import (
	"strings"
	"testing"
)

func TestParseBoardIDValid(t *testing.T) {
	want := InitialBoard(2)
	got, err := ParseBoardID(want.ID())
	if err != nil {
		t.Fatalf("ParseBoardID(%q) returned error: %v", want.ID(), err)
	}
	if got.Size != want.Size {
		t.Fatalf("Size = %d, want %d", got.Size, want.Size)
	}
	if got.Dimension != want.Dimension {
		t.Fatalf("Dimension = %d, want %d", got.Dimension, want.Dimension)
	}
	if got.Turn != want.Turn {
		t.Fatalf("Turn = %d, want %d", got.Turn, want.Turn)
	}
	if string(got.Cells) != string(want.Cells) {
		t.Fatalf("Cells = %v, want %v", got.Cells, want.Cells)
	}
	if got.BoardStatus() != want.BoardStatus() {
		t.Fatalf("BoardStatus() = %q, want %q", got.BoardStatus(), want.BoardStatus())
	}
}

func TestInitialBoardHasDimension(t *testing.T) {
	for _, tt := range []struct {
		size int
		want int
	}{
		{size: 2, want: 4},
		{size: 3, want: 6},
		{size: 4, want: 8},
	} {
		if got := InitialBoard(tt.size).Dimension; got != tt.want {
			t.Fatalf("InitialBoard(%d).Dimension = %d, want %d", tt.size, got, tt.want)
		}
	}
}

func TestBoardStatusMethod(t *testing.T) {
	if got := InitialBoard().BoardStatus(); got != "黒 2、白 2" {
		t.Fatalf("InitialBoard().BoardStatus() = %q, want %q", got, "黒 2、白 2")
	}
}

func TestParseBoardIDAcceptsValidIDs(t *testing.T) {
	tests := []string{
		InitialBoard().ID(),
		"2-2-" + strings.Repeat("1", 16),
		"2-3-" + strings.Repeat("2", 16),
		"4-1-" + strings.Repeat("0", 64),
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
		{name: "payload length mismatch", id: "2-1-0"},
		{name: "malformed cell", id: "2-1-zzzzzzzzzzzzzzzz"},
		{name: "cell value out of range", id: "2-1-" + strings.Repeat("3", 16)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseBoardID(tt.id); err == nil {
				t.Fatalf("ParseBoardID(%q) = nil error, want invalid board ID error", tt.id)
			}
		})
	}
}
