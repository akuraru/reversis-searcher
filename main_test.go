package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBoardHandler(t *testing.T) {
	initial := initialBoard()
	req := httptest.NewRequest(http.MethodGet, "/boards/"+initial.id(), nil)
	res := httptest.NewRecorder()

	newHandler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", res.Code, http.StatusOK)
	}

	body := res.Body.String()
	for _, want := range []string{"A", "B", "C", "D", "1", "2", "3", "4", "黒", "盤面ID"} {
		if !strings.Contains(body, want) {
			t.Errorf("body does not contain %q: %s", want, body)
		}
	}
	if got := strings.Count(body, "href=\"/boards/"); got != 4 {
		t.Fatalf("legal move links = %d, want 4", got)
	}
}

func TestBoardHandlerServesNextBoard(t *testing.T) {
	initial := initialBoard()
	res := httptest.NewRecorder()
	newHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/boards/"+initial.id(), nil))

	const prefix = `href="/boards/`
	body := res.Body.String()
	start := strings.Index(body, prefix)
	if start < 0 {
		t.Fatal("response does not contain a legal move link")
	}
	start += len(prefix)
	end := strings.IndexByte(body[start:], '"')
	if end < 0 {
		t.Fatal("legal move link is not closed")
	}
	nextID := body[start : start+end]

	nextRes := httptest.NewRecorder()
	newHandler().ServeHTTP(nextRes, httptest.NewRequest(http.MethodGet, "/boards/"+nextID, nil))
	if nextRes.Code != http.StatusOK {
		t.Fatalf("next board status = %d, want %d", nextRes.Code, http.StatusOK)
	}
	if !strings.Contains(nextRes.Body.String(), "手番: 白") {
		t.Fatal("next board does not show the changed turn")
	}
}

func TestBoardHandlerErrors(t *testing.T) {
	tests := []struct {
		name string
		path string
		want int
	}{
		{name: "invalid id", path: "/boards/not-an-id", want: http.StatusBadRequest},
		{name: "unknown board", path: "/boards/2-1-0000000000000000", want: http.StatusNotFound},
		{name: "unknown path", path: "/", want: http.StatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := httptest.NewRecorder()
			newHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, test.path, nil))
			if res.Code != test.want {
				t.Fatalf("status code = %d, want %d", res.Code, test.want)
			}
		})
	}
}

func TestBoardHandlerMethodAndHead(t *testing.T) {
	initial := initialBoard()
	path := fmt.Sprintf("/boards/%s", initial.id())

	methodRes := httptest.NewRecorder()
	newHandler().ServeHTTP(methodRes, httptest.NewRequest(http.MethodPost, path, nil))
	if methodRes.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want %d", methodRes.Code, http.StatusMethodNotAllowed)
	}

	headRes := httptest.NewRecorder()
	newHandler().ServeHTTP(headRes, httptest.NewRequest(http.MethodHead, path, nil))
	if headRes.Code != http.StatusOK {
		t.Fatalf("HEAD status = %d, want %d", headRes.Code, http.StatusOK)
	}
	if headRes.Body.Len() != 0 {
		t.Fatalf("HEAD body length = %d, want 0", headRes.Body.Len())
	}
}

func TestLegalNextBoardsInitialPosition(t *testing.T) {
	moves := legalNextBoards(initialBoard())
	if len(moves) != 4 {
		t.Fatalf("initial legal moves = %d, want 4", len(moves))
	}
	for _, move := range moves {
		if move.turn != whiteCell {
			t.Errorf("next turn = %d, want %d", move.turn, whiteCell)
		}
	}
}

func TestLegalNextBoardsFlipsMultipleDirections(t *testing.T) {
	cells := make([]byte, 16)
	cells[2], cells[8], cells[10] = blackCell, blackCell, blackCell
	cells[1], cells[4], cells[5] = whiteCell, whiteCell, whiteCell
	current := board{size: 2, turn: blackCell, cells: cells, status: boardStatus(cells)}

	moves := legalNextBoards(current)
	if len(moves) != 1 {
		t.Fatalf("legal moves = %d, want 1", len(moves))
	}
	want := []byte{blackCell, blackCell, blackCell, emptyCell, blackCell, blackCell, emptyCell, emptyCell, blackCell, emptyCell, blackCell, emptyCell, emptyCell, emptyCell, emptyCell, emptyCell}
	if got := moves[0].cells; !bytes.Equal(got, want) {
		t.Errorf("cells = %v, want %v", got, want)
	}
}

func TestLegalNextBoardsSkipsTurnWhenNoMoveExists(t *testing.T) {
	cells := []byte{
		blackCell, blackCell, blackCell, blackCell,
		blackCell, blackCell, blackCell, blackCell,
		blackCell, blackCell, blackCell, whiteCell,
		blackCell, blackCell, emptyCell, emptyCell,
	}
	current := board{size: 2, turn: whiteCell, cells: cells, status: boardStatus(cells)}

	moves := legalNextBoards(current)
	if len(moves) != 1 {
		t.Fatalf("pass transitions = %d, want 1", len(moves))
	}
	if moves[0].turn != blackCell || !bytes.Equal(moves[0].cells, cells) {
		t.Fatalf("pass transition = %#v, want unchanged board with black turn", moves[0])
	}
}

func TestLegalNextBoardsEndsGameWhenNeitherPlayerCanMove(t *testing.T) {
	cells := bytes.Repeat([]byte{blackCell}, 16)
	current := board{size: 2, turn: whiteCell, cells: cells, status: boardStatus(cells)}

	moves := legalNextBoards(current)
	if len(moves) != 1 {
		t.Fatalf("terminal transitions = %d, want 1", len(moves))
	}
	if moves[0].turn != 3 || !bytes.Equal(moves[0].cells, cells) {
		t.Fatalf("terminal transition = %#v, want unchanged board with terminal turn", moves[0])
	}
}
