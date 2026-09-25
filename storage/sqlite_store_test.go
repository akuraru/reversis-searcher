package storage

import (
	"database/sql"
	"errors"
	"testing"

	"reversis-searcher/board"
)

func newTestSQLiteStore(t *testing.T) *SQLiteBoardStore {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewSQLiteBoardStore(db)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestSQLiteBoardStoreSavesAndReadsBoards(t *testing.T) {
	store := newTestSQLiteStore(t)
	initial := board.InitialBoard(2)

	if err := store.SaveBoard(initial); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveBoard(initial); err != nil {
		t.Fatalf("duplicate board should be ignored: %v", err)
	}

	got, err := store.Get(initial.ID())
	if err != nil {
		t.Fatal(err)
	}
	if got.ID() != initial.ID() || got.BoardStatus() != initial.BoardStatus() {
		t.Fatalf("got board %#v, want %#v", got, initial)
	}
}

func TestSQLiteBoardStoreSavesAndListsTransitions(t *testing.T) {
	store := newTestSQLiteStore(t)
	initial := board.InitialBoard(2)
	next := initial.LegalNextBoards()
	for _, current := range append([]board.Board{initial}, next...) {
		if err := store.SaveBoard(current); err != nil {
			t.Fatal(err)
		}
	}
	for _, current := range next {
		if err := store.SaveTransition(initial.ID(), current.ID()); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SaveTransition(initial.ID(), next[0].ID()); err != nil {
		t.Fatalf("duplicate transition should be ignored: %v", err)
	}

	got, err := store.GetNextBoards(initial.ID())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(next) {
		t.Fatalf("next boards = %d, want %d", len(got), len(next))
	}
	for index := range got {
		if got[index].ID() != next[index].ID() {
			t.Errorf("next board %d = %s, want %s", index, got[index].ID(), next[index].ID())
		}
	}
}

func TestSQLiteBoardStoreRejectsInvalidTransitionAndMissingBoard(t *testing.T) {
	store := newTestSQLiteStore(t)
	initial := board.InitialBoard(2)
	next := initial.LegalNextBoards()[0]
	if err := store.SaveBoard(initial); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveBoard(next); err != nil {
		t.Fatal(err)
	}

	if err := store.SaveTransition(initial.ID(), initial.ID()); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("same board transition error = %v, want %v", err, ErrInvalidTransition)
	}
	if _, err := store.Get("2-1-0000000000000000"); !errors.Is(err, board.ErrBoardNotFound) {
		t.Fatalf("missing board error = %v, want %v", err, board.ErrBoardNotFound)
	}
}

func TestSQLiteBoardStoreSeedsInitialPositions(t *testing.T) {
	store := newTestSQLiteStore(t)
	if err := store.SeedInitialBoard(); err != nil {
		t.Fatal(err)
	}
	if err := store.SeedInitialBoard(); err != nil {
		t.Fatalf("seeding twice should be idempotent: %v", err)
	}

	for _, initial := range board.InitialBoards() {
		boards, err := store.GetNextBoards(initial.ID())
		if err != nil {
			t.Fatal(err)
		}
		if len(boards) != 4 {
			t.Fatalf("size %d seeded next boards = %d, want 4", initial.Size, len(boards))
		}
	}
}
