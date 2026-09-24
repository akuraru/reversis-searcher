package main

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
