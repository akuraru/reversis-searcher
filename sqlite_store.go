package main

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

const boardSchema = `
CREATE TABLE IF NOT EXISTS boards (
    id TEXT PRIMARY KEY,
    size INTEGER NOT NULL CHECK (size IN (2, 3, 4)),
    turn INTEGER NOT NULL CHECK (turn IN (1, 2, 3)),
    cells BLOB NOT NULL,
    status TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS transitions (
    parent_id TEXT NOT NULL REFERENCES boards(id),
    child_id TEXT NOT NULL REFERENCES boards(id),
    move INTEGER NOT NULL CHECK (move >= 0),
    PRIMARY KEY (parent_id, child_id),
    UNIQUE (parent_id, move)
);`

var errInvalidTransition = errors.New("invalid board transition")

type sqliteBoardStore struct {
	db *sql.DB
}

func newSQLiteBoardStore(db *sql.DB) (*sqliteBoardStore, error) {
	if db == nil {
		return nil, errors.New("nil database")
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}
	if _, err := db.Exec(boardSchema); err != nil {
		return nil, err
	}
	return &sqliteBoardStore{db: db}, nil
}

func (s *sqliteBoardStore) SaveBoard(b board) error {
	if parsed, err := parseBoardID(b.id()); err != nil || parsed.id() != b.id() {
		return errInvalidBoardID
	}
	if b.status != boardStatus(b.cells) {
		return errInvalidBoardID
	}
	_, err := s.db.Exec(
		`INSERT INTO boards (id, size, turn, cells, status) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO NOTHING`,
		b.id(), b.size, b.turn, b.cells, b.status,
	)
	return err
}

func (s *sqliteBoardStore) SaveTransition(parentID, childID string) error {
	parent, err := s.Get(parentID)
	if err != nil {
		return err
	}
	child, err := s.Get(childID)
	if err != nil {
		return err
	}
	move := -1
	for _, next := range legalNextBoards(parent) {
		if next.id() == child.id() {
			for index, cell := range next.cells {
				if parent.cells[index] == emptyCell && cell != emptyCell {
					move = index
					break
				}
			}
			break
		}
	}
	if move < 0 {
		return errInvalidTransition
	}
	_, err = s.db.Exec(
		`INSERT INTO transitions (parent_id, child_id, move) VALUES (?, ?, ?)
		 ON CONFLICT(parent_id, child_id) DO NOTHING`,
		parent.id(), child.id(), move,
	)
	return err
}

func (s *sqliteBoardStore) Get(id string) (board, error) {
	if _, err := parseBoardID(id); err != nil {
		return board{}, errInvalidBoardID
	}
	var b board
	var cells []byte
	err := s.db.QueryRow(
		`SELECT size, turn, cells, status FROM boards WHERE id = ?`, id,
	).Scan(&b.size, &b.turn, &cells, &b.status)
	if errors.Is(err, sql.ErrNoRows) {
		return board{}, errBoardNotFound
	}
	if err != nil {
		return board{}, err
	}
	b.cells = append([]byte(nil), cells...)
	if b.id() != id {
		return board{}, fmt.Errorf("stored board %q has inconsistent data", id)
	}
	return b, nil
}

func (s *sqliteBoardStore) GetNextBoards(parentID string) ([]board, error) {
	if _, err := s.Get(parentID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`
		SELECT b.size, b.turn, b.cells, b.status
		FROM transitions t JOIN boards b ON b.id = t.child_id
		WHERE t.parent_id = ? ORDER BY t.move`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var boards []board
	for rows.Next() {
		var b board
		var cells []byte
		if err := rows.Scan(&b.size, &b.turn, &cells, &b.status); err != nil {
			return nil, err
		}
		b.cells = append([]byte(nil), cells...)
		boards = append(boards, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return boards, nil
}

func (s *sqliteBoardStore) SeedInitialBoard() error {
	initial := initialBoard()
	if err := s.SaveBoard(initial); err != nil {
		return err
	}
	for _, next := range legalNextBoards(initial) {
		if err := s.SaveBoard(next); err != nil {
			return err
		}
		if err := s.SaveTransition(initial.id(), next.id()); err != nil {
			return err
		}
	}
	return nil
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
	if err != nil || (turn != 1 && turn != 2 && turn != 3) || len(parts[2]) != dimension*dimension*2 {
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
		var flips []int
		for _, direction := range directions {
			r, c := row+direction[0], column+direction[1]
			var line []int
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
