package storage

import (
	"database/sql"
	"errors"
	"fmt"

	"reversis-searcher/board"

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

var ErrInvalidTransition = errors.New("invalid board transition")

type SQLiteBoardStore struct {
	db *sql.DB
}

func NewSQLiteBoardStore(db *sql.DB) (*SQLiteBoardStore, error) {
	if db == nil {
		return nil, errors.New("nil database")
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}
	if _, err := db.Exec(boardSchema); err != nil {
		return nil, err
	}
	return &SQLiteBoardStore{db: db}, nil
}

func (s *SQLiteBoardStore) SaveBoard(b board.Board) error {
	parsed, err := board.ParseBoardID(b.ID())
	if err != nil || parsed.ID() != b.ID() {
		return board.ErrInvalidBoardID
	}
	cells := make([]byte, len(b.Cells))
	for index, cell := range b.Cells {
		cells[index] = byte(cell)
	}
	_, err = s.db.Exec(
		`INSERT INTO boards (id, size, turn, cells, status) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO NOTHING`,
		b.ID(), b.Size, b.Turn, cells, b.BoardStatus(),
	)
	return err
}

func (s *SQLiteBoardStore) SaveTransition(parentID, childID string) error {
	parent, err := s.Get(parentID)
	if err != nil {
		return err
	}
	child, err := s.Get(childID)
	if err != nil {
		return err
	}
	move := -1
	for _, next := range parent.LegalNextBoards() {
		if next.ID() != child.ID() {
			continue
		}
		for index, cell := range next.Cells {
			if parent.Cells[index] == board.EmptyCell && cell != board.EmptyCell {
				move = index
				break
			}
		}
		break
	}
	if move < 0 {
		return ErrInvalidTransition
	}
	_, err = s.db.Exec(
		`INSERT INTO transitions (parent_id, child_id, move) VALUES (?, ?, ?)
		 ON CONFLICT(parent_id, child_id) DO NOTHING`,
		parent.ID(), child.ID(), move,
	)
	return err
}

func (s *SQLiteBoardStore) Get(id string) (board.Board, error) {
	if _, err := board.ParseBoardID(id); err != nil {
		return board.Board{}, board.ErrInvalidBoardID
	}
	var stored board.Board
	var cells []byte
	var status string
	err := s.db.QueryRow(
		`SELECT size, turn, cells, status FROM boards WHERE id = ?`, id,
	).Scan(&stored.Size, &stored.Turn, &cells, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return board.Board{}, board.ErrBoardNotFound
	}
	if err != nil {
		return board.Board{}, err
	}
	stored.Dimension, _ = board.BoardDimension(stored.Size)
	stored.Cells = make([]board.Cell, len(cells))
	for index, cell := range cells {
		stored.Cells[index] = board.Cell(cell)
	}
	if stored.ID() != id || stored.BoardStatus() != status {
		return board.Board{}, fmt.Errorf("stored board %q has inconsistent data", id)
	}
	return stored, nil
}

func (s *SQLiteBoardStore) GetNextBoards(parentID string) ([]board.Board, error) {
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

	var boards []board.Board
	for rows.Next() {
		var next board.Board
		var cells []byte
		var status string
		if err := rows.Scan(&next.Size, &next.Turn, &cells, &status); err != nil {
			return nil, err
		}
		next.Dimension, _ = board.BoardDimension(next.Size)
		next.Cells = make([]board.Cell, len(cells))
		for index, cell := range cells {
			next.Cells[index] = board.Cell(cell)
		}
		if next.BoardStatus() != status {
			return nil, fmt.Errorf("stored board %q has inconsistent status", next.ID())
		}
		boards = append(boards, next)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return boards, nil
}

func (s *SQLiteBoardStore) SeedInitialBoard() error {
	for _, initial := range board.InitialBoards() {
		if err := s.SaveBoard(initial); err != nil {
			return err
		}
		for _, next := range initial.LegalNextBoards() {
			if err := s.SaveBoard(next); err != nil {
				return err
			}
			if err := s.SaveTransition(initial.ID(), next.ID()); err != nil {
				return err
			}
		}
	}
	return nil
}
