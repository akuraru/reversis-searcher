package main

import (
	"log"
	"net/http"

	boardpkg "reversis-searcher/board"
	handlerspkg "reversis-searcher/handlers"
	storagepkg "reversis-searcher/storage"
)

func newHandler() http.Handler {
	return newHandlerWithStore(storagepkg.NewMemoryBoardStore())
}

func newHandlerWithStore(store boardpkg.Store) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/boards/", handlerspkg.NewBoardHandler(store))
	mux.Handle("/", handlerspkg.NewIndexHandler())
	return mux
}

func main() {
	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", newHandler()); err != nil {
		log.Fatal(err)
	}
}
