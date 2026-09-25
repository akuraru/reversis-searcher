package storage

import "reversis-searcher/board"

type MemoryBoardStore struct {
	boards map[string]board.Board
}

func NewMemoryBoardStore() *MemoryBoardStore {
	initial := board.InitialBoard(2)
	store := &MemoryBoardStore{boards: map[string]board.Board{initial.ID(): initial}}
	for _, next := range initial.LegalNextBoards() {
		store.boards[next.ID()] = next
	}
	return store
}

func (s *MemoryBoardStore) Get(id string) (board.Board, error) {
	if _, err := board.ParseBoardID(id); err != nil {
		return board.Board{}, board.ErrInvalidBoardID
	}
	b, ok := s.boards[id]
	if !ok {
		return board.Board{}, board.ErrBoardNotFound
	}
	return b, nil
}
