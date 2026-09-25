package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"reversis-searcher/board"
)

type BoardPage struct {
	ID      string
	Turn    board.Turn
	Status  string
	Columns []string
	Rows    []BoardRow
}

type BoardRow struct {
	Number int
	Cells  []BoardCell
}

type BoardCell struct {
	Value board.Cell
	Move  string
}

func NewBoardPage(b board.Board) BoardPage {
	dimension := b.Dimension
	columns := make([]string, dimension)
	for i := range columns {
		columns[i] = string(rune('A' + i))
	}
	moves := board.LegalNextBoards(b)
	moveByIndex := make(map[int]string)
	for _, move := range moves {
		for index, cell := range move.Cells {
			if b.Cells[index] == board.EmptyCell && cell != board.EmptyCell {
				moveByIndex[index] = move.ID()
				break
			}
		}
	}
	rows := make([]BoardRow, dimension)
	for row := range rows {
		rows[row].Number = row + 1
		rows[row].Cells = make([]BoardCell, dimension)
		for column := range rows[row].Cells {
			index := row*dimension + column
			rows[row].Cells[column] = BoardCell{Value: b.Cells[index], Move: moveByIndex[index]}
		}
	}
	return BoardPage{ID: b.ID(), Turn: b.Turn, Status: b.Status, Columns: columns, Rows: rows}
}

var BoardPageTemplate = template.Must(template.New("board").Parse(`<!doctype html>
<html lang="ja">
<meta charset="utf-8">
<title>リバーシ {{.ID}}</title>

<h1>リバーシ盤面</h1>
<p>盤面ID: <code>{{.ID}}</code></p>
<p>手番: {{if eq .Turn 1}}黒{{else if eq .Turn 2}}白{{else}}決着{{end}}</p>
<p>状態: {{.Status}}</p>
<table>
<thead><tr><th></th>{{range .Columns}}<th>{{.}}</th>{{end}}</tr></thead>
<tbody>{{range .Rows}}<tr><th>{{.Number}}</th>{{range .Cells}}<td>{{if .Move}}<a href="/boards/{{.Move}}">{{end}}{{if eq .Value 1}}●{{else if eq .Value 2}}○{{else}}・{{end}}{{if .Move}}</a>{{end}}</td>{{end}}</tr>{{end}}</tbody>
</table>
</html>
`))

func WriteBoardPage(w http.ResponseWriter, r *http.Request, b board.Board) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page := NewBoardPage(b)
	var body bytes.Buffer
	if err := BoardPageTemplate.Execute(&body, page); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", body.Len()))
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(body.Bytes())
}
