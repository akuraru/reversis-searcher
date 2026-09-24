package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

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
	dimension, _ := boardDimension(current.size)
	if current.turn == 3 {
		return nil
	}
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
