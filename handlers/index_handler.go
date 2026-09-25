package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"reversis-searcher/board"
)

func NewIndexHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		WriteIndexPage(w, r, board.InitialBoards())
	})
}

func WriteIndexPage(w http.ResponseWriter, r *http.Request, boards []board.Board) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var body bytes.Buffer
	if err := IndexPageTemplate.Execute(&body, boards); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", body.Len()))
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(body.Bytes())
}

var IndexPageTemplate = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="ja">
<meta charset="utf-8">
<title>リバーシ</title>

<h1>リバーシ</h1>
<ul>{{range .}}<li><a href="/boards/{{.ID}}">サイズ {{.Size}}</a></li>{{end}}</ul>
</html>
`))
