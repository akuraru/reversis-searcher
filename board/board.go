package board

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Cell byte

const (
	EmptyCell Cell = iota
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
	Size      int
	Dimension int
	Turn      Turn
	Cells     []Cell
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

func InitialBoard(targetSize int) Board {
	dimension, ok := BoardDimension(targetSize)
	if !ok {
		return Board{}
	}
	cells := make([]Cell, dimension*dimension)
	center := dimension / 2
	cells[(center-1)*dimension+(center-1)] = WhiteCell
	cells[(center-1)*dimension+center] = BlackCell
	cells[center*dimension+(center-1)] = BlackCell
	cells[center*dimension+center] = WhiteCell
	return Board{Size: targetSize, Dimension: dimension, Turn: BlackTurn, Cells: cells}
}

func InitialBoards() []Board {
	return []Board{InitialBoard(2), InitialBoard(3), InitialBoard(4)}
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
	if len(parts[2]) != dimension*dimension {
		return Board{}, ErrInvalidBoardID
	}
	cells := make([]Cell, len(parts[2]))
	for index, cell := range []byte(parts[2]) {
		if cell < '0' || cell > '2' {
			return Board{}, ErrInvalidBoardID
		}
		cells[index] = Cell(cell - '0')
	}
	for _, cell := range cells {
		if cell > WhiteCell {
			return Board{}, ErrInvalidBoardID
		}
	}
	return Board{Size: size, Dimension: dimension, Turn: Turn(turn), Cells: cells}, nil
}

func (b Board) ID() string {
	cells := make([]byte, len(b.Cells))
	for index, cell := range b.Cells {
		cells[index] = '0' + byte(cell)
	}
	return fmt.Sprintf("%d-%d-%s", b.Size, b.Turn, cells)
}

func (b Board) BoardStatus() string {
	black, white := 0, 0
	for _, cell := range b.Cells {
		switch cell {
		case BlackCell:
			black++
		case WhiteCell:
			white++
		}
	}
	return fmt.Sprintf("黒 %d、白 %d", black, white)
}

func (b Board) LegalNextBoards() []Board {
	if b.Turn == FinishedTurn {
		return nil
	}
	moves := b.legalMoveBoards()
	if len(moves) > 0 {
		return moves
	}
	passed := b
	passed.Turn = Turn(3 - b.Turn)
	if len(passed.legalMoveBoards()) > 0 {
		return []Board{passed}
	}
	ended := b
	ended.Turn = FinishedTurn
	return []Board{ended}
}

func (b Board) legalMoveBoards() []Board {
	dimension := b.Dimension
	stone := Cell(b.Turn)
	opponent := Turn(3 - b.Turn)
	opponentCell := Cell(opponent)
	directions := [][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}}
	var nextBoards []Board
	for index, cell := range b.Cells {
		if cell != EmptyCell {
			continue
		}
		row, column := index/dimension, index%dimension
		flips := []int{}
		for _, direction := range directions {
			r, c := row+direction[0], column+direction[1]
			line := []int{}
			for r >= 0 && r < dimension && c >= 0 && c < dimension && b.Cells[r*dimension+c] == opponentCell {
				line = append(line, r*dimension+c)
				r += direction[0]
				c += direction[1]
			}
			if len(line) > 0 && r >= 0 && r < dimension && c >= 0 && c < dimension && b.Cells[r*dimension+c] == stone {
				flips = append(flips, line...)
			}
		}
		if len(flips) == 0 {
			continue
		}
		cells := append([]Cell(nil), b.Cells...)
		cells[index] = stone
		for _, flip := range flips {
			cells[flip] = stone
		}
		nextBoards = append(nextBoards, Board{Size: b.Size, Dimension: dimension, Turn: opponent, Cells: cells})
	}
	return nextBoards
}
