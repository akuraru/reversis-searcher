package board

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	EmptyCell byte = iota
	BlackCell
	WhiteCell
)

type Turn byte

const (
	BlackTurn Turn = iota + 1
	WhiteTurn
	FinishedTurn
)

type Board struct {
	Size   int
	Turn   Turn
	Cells  []byte
	Status string
}

type Store interface {
	Get(string) (Board, error)
}

var (
	ErrInvalidBoardID = errors.New("invalid board ID")
	ErrBoardNotFound  = errors.New("board not found")
)

func BoardDimension(size int) (int, bool) {
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

func (b Board) ID() string {
	return fmt.Sprintf("%d-%d-%s", b.Size, b.Turn, hex.EncodeToString(b.Cells))
}

func ParseBoardID(id string) (Board, error) {
	parts := strings.Split(id, "-")
	if len(parts) != 3 {
		return Board{}, ErrInvalidBoardID
	}
	size, err := strconv.Atoi(parts[0])
	if err != nil {
		return Board{}, ErrInvalidBoardID
	}
	dimension, ok := BoardDimension(size)
	if !ok {
		return Board{}, ErrInvalidBoardID
	}
	turn, err := strconv.Atoi(parts[1])
	if err != nil || (turn != 1 && turn != 2 && turn != 3) {
		return Board{}, ErrInvalidBoardID
	}
	if len(parts[2]) != dimension*dimension*2 {
		return Board{}, ErrInvalidBoardID
	}
	cells, err := hex.DecodeString(parts[2])
	if err != nil {
		return Board{}, ErrInvalidBoardID
	}
	for _, cell := range cells {
		if cell > WhiteCell {
			return Board{}, ErrInvalidBoardID
		}
	}
	return Board{Size: size, Turn: Turn(turn), Cells: cells, Status: BoardStatus(cells)}, nil
}

func BoardStatus(cells []byte) string {
	black, white := 0, 0
	for _, cell := range cells {
		switch cell {
		case BlackCell:
			black++
		case WhiteCell:
			white++
		}
	}
	return fmt.Sprintf("黒 %d、白 %d", black, white)
}

func InitialBoard() Board {
	cells := make([]byte, 16)
	cells[5], cells[6] = WhiteCell, BlackCell
	cells[9], cells[10] = BlackCell, WhiteCell
	return Board{Size: 2, Turn: BlackTurn, Cells: cells, Status: BoardStatus(cells)}
}

func LegalNextBoards(current Board) []Board {
	if current.Turn == FinishedTurn {
		return nil
	}
	moves := legalMoveBoards(current)
	if len(moves) > 0 {
		return moves
	}
	passed := current
	passed.Turn = Turn(3 - current.Turn)
	if len(legalMoveBoards(passed)) > 0 {
		return []Board{passed}
	}
	ended := current
	ended.Turn = FinishedTurn
	return []Board{ended}
}

func legalMoveBoards(current Board) []Board {
	dimension, _ := BoardDimension(current.Size)
	stone := byte(current.Turn)
	opponent := Turn(3 - current.Turn)
	directions := [][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}}
	var nextBoards []Board
	for index, cell := range current.Cells {
		if cell != EmptyCell {
			continue
		}
		row, column := index/dimension, index%dimension
		flips := []int{}
		for _, direction := range directions {
			r, c := row+direction[0], column+direction[1]
			line := []int{}
			for r >= 0 && r < dimension && c >= 0 && c < dimension && current.Cells[r*dimension+c] == byte(opponent) {
				line = append(line, r*dimension+c)
				r += direction[0]
				c += direction[1]
			}
			if len(line) > 0 && r >= 0 && r < dimension && c >= 0 && c < dimension && current.Cells[r*dimension+c] == stone {
				flips = append(flips, line...)
			}
		}
		if len(flips) == 0 {
			continue
		}
		cells := append([]byte(nil), current.Cells...)
		cells[index] = stone
		for _, flip := range flips {
			cells[flip] = stone
		}
		nextBoards = append(nextBoards, Board{Size: current.Size, Turn: opponent, Cells: cells, Status: BoardStatus(cells)})
	}
	return nextBoards
}
