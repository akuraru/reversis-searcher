package main

import (
	"database/sql"
	"errors"
	"fmt"

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
