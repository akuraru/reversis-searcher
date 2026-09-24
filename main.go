package main

import (
	"errors"
	"log"
	"net/http"
	"strings"

	boardpkg "reversis-searcher/board"
	handlerspkg "reversis-searcher/handlers"
	storagepkg "reversis-searcher/storage"
)

func newHandler() http.Handler {
	return newHandlerWithStore(storagepkg.NewMemoryBoardStore())
}

func newHandlerWithStore(store boardpkg.Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || !strings.HasPrefix(r.URL.Path, "/boards/") {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/boards/")
		if id == "" || strings.Contains(id, "/") {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		current, err := store.Get(id)
		if errors.Is(err, boardpkg.ErrInvalidBoardID) {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		if errors.Is(err, boardpkg.ErrBoardNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		handlerspkg.WriteBoardPage(w, r, current)
	})
	return mux
}

func main() {
	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", newHandler()); err != nil {
		log.Fatal(err)
	}
}
