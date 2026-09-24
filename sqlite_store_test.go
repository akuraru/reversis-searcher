package main

import (
	"database/sql"
	"errors"
	"testing"
)

func newTestSQLiteStore(t *testing.T) *sqliteBoardStore {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := newSQLiteBoardStore(db)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestSQLiteBoardStoreSavesAndReadsBoards(t *testing.T) {
	store := newTestSQLiteStore(t)
	initial := initialBoard()

	if err := store.SaveBoard(initial); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveBoard(initial); err != nil {
		t.Fatalf("duplicate board should be ignored: %v", err)
	}

	got, err := store.Get(initial.id())
	if err != nil {
		t.Fatal(err)
	}
	if got.id() != initial.id() || got.status != initial.status {
		t.Fatalf("got board %#v, want %#v", got, initial)
	}
}

func TestSQLiteBoardStoreSavesAndListsTransitions(t *testing.T) {
	store := newTestSQLiteStore(t)
	initial := initialBoard()
	next := legalNextBoards(initial)
	for _, b := range append([]board{initial}, next...) {
		if err := store.SaveBoard(b); err != nil {
			t.Fatal(err)
		}
	}
	for _, b := range next {
		if err := store.SaveTransition(initial.id(), b.id()); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SaveTransition(initial.id(), next[0].id()); err != nil {
		t.Fatalf("duplicate transition should be ignored: %v", err)
	}

	got, err := store.GetNextBoards(initial.id())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(next) {
		t.Fatalf("next boards = %d, want %d", len(got), len(next))
	}
	for i := range got {
		if got[i].id() != next[i].id() {
			t.Errorf("next board %d = %s, want %s", i, got[i].id(), next[i].id())
		}
	}
}

func TestSQLiteBoardStoreRejectsInvalidTransitionAndMissingBoard(t *testing.T) {
	store := newTestSQLiteStore(t)
	initial := initialBoard()
	next := legalNextBoards(initial)[0]
	if err := store.SaveBoard(initial); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveBoard(next); err != nil {
		t.Fatal(err)
	}

	if err := store.SaveTransition(initial.id(), initial.id()); !errors.Is(err, errInvalidTransition) {
		t.Fatalf("same board transition error = %v, want %v", err, errInvalidTransition)
	}
	if _, err := store.Get("2-1-00000000000000000000000000000000"); !errors.Is(err, errBoardNotFound) {
		t.Fatalf("missing board error = %v, want %v", err, errBoardNotFound)
	}
}

func TestSQLiteBoardStoreSeedsInitialPosition(t *testing.T) {
	store := newTestSQLiteStore(t)
	if err := store.SeedInitialBoard(); err != nil {
		t.Fatal(err)
	}
	if err := store.SeedInitialBoard(); err != nil {
		t.Fatalf("seeding twice should be idempotent: %v", err)
	}

	boards, err := store.GetNextBoards(initialBoard().id())
	if err != nil {
		t.Fatal(err)
	}
	if len(boards) != 4 {
		t.Fatalf("seeded next boards = %d, want 4", len(boards))
	}
}
