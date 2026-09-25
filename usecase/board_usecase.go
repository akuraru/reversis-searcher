package usecase

import "reversis-searcher/board"

type BoardUseCase struct {
	store board.Store
}

func NewBoardUseCase(store board.Store) *BoardUseCase {
	return &BoardUseCase{store: store}
}

func (u *BoardUseCase) GetBoard(id string) (board.Board, error) {
	return u.store.Get(id)
}
