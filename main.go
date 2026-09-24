package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func newHandler() http.Handler {
	return newHandlerWithStore(newMemoryBoardStore())
}

func newHandlerWithStore(store boardStore) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || !strings.HasPrefix(r.URL.Path, "/boards/") {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/boards/")
		if id == "" || strings.Contains(id, "/") {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		current, err := store.Get(id)
		if errors.Is(err, errInvalidBoardID) {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		if errors.Is(err, errBoardNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		page := newBoardPage(current)
		var body bytes.Buffer
		if err := boardPageTemplate.Execute(&body, page); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", body.Len()))
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write(body.Bytes())
	})
	return mux
}

type boardPage struct {
	ID      string
	Turn    byte
	Status  string
	Columns []string
	Rows    []boardRow
}

type boardRow struct {
	Number int
	Cells  []boardCell
}

type boardCell struct {
	Value byte
	Move  string
}

func newBoardPage(b board) boardPage {
	dimension, _ := boardDimension(b.size)
	columns := make([]string, dimension)
	for i := range columns {
		columns[i] = string(rune('A' + i))
	}
	moves := legalNextBoards(b)
	moveByIndex := make(map[int]string)
	for _, move := range moves {
		for index, cell := range move.cells {
			if b.cells[index] == emptyCell && cell != emptyCell {
				moveByIndex[index] = move.id()
				break
			}
		}
	}
	rows := make([]boardRow, dimension)
	for row := range rows {
		rows[row].Number = row + 1
		rows[row].Cells = make([]boardCell, dimension)
		for column := range rows[row].Cells {
			index := row*dimension + column
			rows[row].Cells[column] = boardCell{Value: b.cells[index], Move: moveByIndex[index]}
		}
	}
	return boardPage{ID: b.id(), Turn: b.turn, Status: b.status, Columns: columns, Rows: rows}
}

var boardPageTemplate = template.Must(template.New("board").Parse(`<!doctype html>
<html lang="ja">
<head><meta charset="utf-8"><title>リバーシ {{.ID}}</title></head>
<body>
<h1>リバーシ盤面</h1>
<p>盤面ID: <code>{{.ID}}</code></p>
<p>手番: {{if eq .Turn 1}}黒{{else if eq .Turn 2}}白{{else}}決着{{end}}</p>
<p>状態: {{.Status}}</p>
<table>
<thead><tr><th></th>{{range .Columns}}<th>{{.}}</th>{{end}}</tr></thead>
<tbody>{{range .Rows}}<tr><th>{{.Number}}</th>{{range .Cells}}<td>{{if .Move}}<a href="/boards/{{.Move}}">{{end}}{{if eq .Value 1}}●{{else if eq .Value 2}}○{{else}}・{{end}}{{if .Move}}</a>{{end}}</td>{{end}}</tr>{{end}}</tbody>
</table>
</body>
</html>`))

func main() {
	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", newHandler()); err != nil {
		log.Fatal(err)
	}
}

const (
	emptyCell byte = iota
	blackCell
	whiteCell
)

type board struct {
	size   int
	turn   byte
	cells  []byte
	status string
}

type boardStore interface {
	Get(string) (board, error)
}

var (
	errInvalidBoardID = errors.New("invalid board ID")
	errBoardNotFound  = errors.New("board not found")
)

func boardDimension(size int) (int, bool) {
	switch size {
	case 2:
		return 4, true
	case 3:
		return 6, true
	case 4:
		return 8, true
	default:
		return 0, false
	}
}

func (b board) id() string {
	return fmt.Sprintf("%d-%d-%s", b.size, b.turn, hex.EncodeToString(b.cells))
}

func parseBoardID(id string) (board, error) {
	parts := strings.Split(id, "-")
	if len(parts) != 3 {
		return board{}, errInvalidBoardID
	}
	size, err := strconv.Atoi(parts[0])
	if err != nil {
		return board{}, errInvalidBoardID
	}
	dimension, ok := boardDimension(size)
	if !ok {
		return board{}, errInvalidBoardID
	}
	turn, err := strconv.Atoi(parts[1])
	if err != nil || (turn != 1 && turn != 2 && turn != 3) {
		return board{}, errInvalidBoardID
	}
	if len(parts[2]) != dimension*dimension*2 {
		return board{}, errInvalidBoardID
	}
	cells, err := hex.DecodeString(parts[2])
	if err != nil {
		return board{}, errInvalidBoardID
	}
	for _, cell := range cells {
		if cell > whiteCell {
			return board{}, errInvalidBoardID
		}
	}
	return board{size: size, turn: byte(turn), cells: cells, status: boardStatus(cells)}, nil
}

func boardStatus(cells []byte) string {
	black, white := 0, 0
	for _, cell := range cells {
		switch cell {
		case blackCell:
			black++
		case whiteCell:
			white++
		}
	}
	return fmt.Sprintf("黒 %d、白 %d", black, white)
}

func initialBoard() board {
	cells := make([]byte, 16)
	cells[5], cells[6] = whiteCell, blackCell
	cells[9], cells[10] = blackCell, whiteCell
	return board{size: 2, turn: 1, cells: cells, status: boardStatus(cells)}
}

type memoryBoardStore struct {
	boards map[string]board
}

func newMemoryBoardStore() *memoryBoardStore {
	initial := initialBoard()
	store := &memoryBoardStore{boards: map[string]board{initial.id(): initial}}
	for _, next := range legalNextBoards(initial) {
		store.boards[next.id()] = next
	}
	return store
}

func (s *memoryBoardStore) Get(id string) (board, error) {
	if _, err := parseBoardID(id); err != nil {
		return board{}, errInvalidBoardID
	}
	b, ok := s.boards[id]
	if !ok {
		return board{}, errBoardNotFound
	}
	return b, nil
}

func legalNextBoards(current board) []board {
	if current.turn == 3 {
		return nil
	}
	moves := legalMoveBoards(current)
	if len(moves) > 0 {
		return moves
	}

	passed := current
	passed.turn = byte(3 - current.turn)
	if len(legalMoveBoards(passed)) > 0 {
		return []board{passed}
	}

	ended := current
	ended.turn = 3
	return []board{ended}
}

func legalMoveBoards(current board) []board {
	dimension, _ := boardDimension(current.size)
	stone := current.turn
	opponent := byte(3 - current.turn)
	directions := [][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}}
	var nextBoards []board
	for index, cell := range current.cells {
		if cell != emptyCell {
			continue
		}
		row, column := index/dimension, index%dimension
		flips := []int{}
		for _, direction := range directions {
			r, c := row+direction[0], column+direction[1]
			line := []int{}
			for r >= 0 && r < dimension && c >= 0 && c < dimension && current.cells[r*dimension+c] == opponent {
				line = append(line, r*dimension+c)
				r += direction[0]
				c += direction[1]
			}
			if len(line) > 0 && r >= 0 && r < dimension && c >= 0 && c < dimension && current.cells[r*dimension+c] == stone {
				flips = append(flips, line...)
			}
		}
		if len(flips) == 0 {
			continue
		}
		cells := append([]byte(nil), current.cells...)
		cells[index] = stone
		for _, flip := range flips {
			cells[flip] = stone
		}
		nextBoards = append(nextBoards, board{size: current.size, turn: opponent, cells: cells, status: boardStatus(cells)})
	}
	return nextBoards
}
