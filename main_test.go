package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBoardHandler(t *testing.T) {
	initial := initialBoard()
	req := httptest.NewRequest(http.MethodGet, "/boards/"+initial.id(), nil)
	res := httptest.NewRecorder()

	newHandler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", res.Code, http.StatusOK)
	}

	body := res.Body.String()
	for _, want := range []string{"A", "B", "C", "D", "1", "2", "3", "4", "黒", "盤面ID"} {
		if !strings.Contains(body, want) {
			t.Errorf("body does not contain %q: %s", want, body)
		}
	}
	if got := strings.Count(body, "href=\"/boards/"); got != 4 {
		t.Fatalf("legal move links = %d, want 4", got)
	}
}

func TestBoardHandlerServesNextBoard(t *testing.T) {
	initial := initialBoard()
	res := httptest.NewRecorder()
	newHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/boards/"+initial.id(), nil))

	const prefix = `href="/boards/`
	body := res.Body.String()
	start := strings.Index(body, prefix)
	if start < 0 {
		t.Fatal("response does not contain a legal move link")
	}
	start += len(prefix)
	end := strings.IndexByte(body[start:], '"')
	if end < 0 {
		t.Fatal("legal move link is not closed")
	}
	nextID := body[start : start+end]

	nextRes := httptest.NewRecorder()
	newHandler().ServeHTTP(nextRes, httptest.NewRequest(http.MethodGet, "/boards/"+nextID, nil))
	if nextRes.Code != http.StatusOK {
		t.Fatalf("next board status = %d, want %d", nextRes.Code, http.StatusOK)
	}
	if !strings.Contains(nextRes.Body.String(), "手番: 白") {
		t.Fatal("next board does not show the changed turn")
	}
}

func TestBoardHandlerErrors(t *testing.T) {
	tests := []struct {
		name string
		path string
		want int
	}{
		{name: "invalid id", path: "/boards/not-an-id", want: http.StatusBadRequest},
		{name: "unknown board", path: "/boards/2-1-00000000000000000000000000000000", want: http.StatusNotFound},
		{name: "unknown path", path: "/", want: http.StatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := httptest.NewRecorder()
			newHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, test.path, nil))
			if res.Code != test.want {
				t.Fatalf("status code = %d, want %d", res.Code, test.want)
			}
		})
	}
}

func TestBoardHandlerMethodAndHead(t *testing.T) {
	initial := initialBoard()
	path := fmt.Sprintf("/boards/%s", initial.id())

	methodRes := httptest.NewRecorder()
	newHandler().ServeHTTP(methodRes, httptest.NewRequest(http.MethodPost, path, nil))
	if methodRes.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want %d", methodRes.Code, http.StatusMethodNotAllowed)
	}

	headRes := httptest.NewRecorder()
	newHandler().ServeHTTP(headRes, httptest.NewRequest(http.MethodHead, path, nil))
	if headRes.Code != http.StatusOK {
		t.Fatalf("HEAD status = %d, want %d", headRes.Code, http.StatusOK)
	}
	if headRes.Body.Len() != 0 {
		t.Fatalf("HEAD body length = %d, want 0", headRes.Body.Len())
	}
}
