package folio

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestSearchRanksBySharedTerms(t *testing.T) {
	idx := New()
	if err := idx.Add("a", "blue whale in the bay"); err != nil {
		t.Fatal(err)
	}
	if err := idx.Add("b", "the bay is quiet"); err != nil {
		t.Fatal(err)
	}
	hits, err := idx.Search("bay whale")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 || hits[0].ID != "a" || hits[0].Terms != 2 || hits[1].ID != "b" || hits[1].Terms != 1 {
		t.Fatalf("%+v", hits)
	}
	if err := idx.Add("a", "quiet shore"); err != nil {
		t.Fatal(err)
	}
	hits, err = idx.Search("whale")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("replaced doc still matched %+v", hits)
	}
	if err := idx.Delete("b"); err != nil {
		t.Fatal(err)
	}
	hits, err = idx.Search("bay")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("deleted doc still matched %+v", hits)
	}
}

func TestRestartDropsATornLine(t *testing.T) {
	dir := t.TempDir()
	idx, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := idx.Add("a", "blue whale"); err != nil {
		t.Fatal(err)
	}
	if err := idx.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(dir+"/index.log", os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{\"op\":\"add\",\"id\":\"b\",\"text\":\"torn"); err != nil {
		t.Fatal(err)
	}
	f.Close()

	idx, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	hits, err := idx.Search("whale")
	if err != nil || len(hits) != 1 || hits[0].ID != "a" {
		t.Fatalf("%+v %v", hits, err)
	}
	if err := idx.Add("c", "quiet bay"); err != nil {
		t.Fatal(err)
	}
	hits, err = idx.Search("bay")
	if err != nil || len(hits) != 1 || hits[0].ID != "c" {
		t.Fatalf("%+v %v", hits, err)
	}
}

func TestHTTPSearch(t *testing.T) {
	idx := New()
	h := Handler(idx)
	put := httptest.NewRequest(http.MethodPut, "/v1/docs/a", strings.NewReader("blue whale in the bay"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, put)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("put %d", rec.Code)
	}
	get := httptest.NewRequest(http.MethodGet, "/v1/search?q=whale", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, get)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"id":"a"`) {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestEmptyQuery(t *testing.T) {
	if _, err := New().Search("the"); err != ErrInvalid {
		t.Fatalf("%v", err)
	}
}

func TestPageAndDocuments(t *testing.T) {
	idx := New()
	if err := idx.Add("a", "blue whale"); err != nil {
		t.Fatal(err)
	}
	h := Handler(idx)
	page := httptest.NewRecorder()
	h.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Hello") {
		t.Fatalf("page %d", page.Code)
	}
	docs := httptest.NewRecorder()
	h.ServeHTTP(docs, httptest.NewRequest(http.MethodGet, "/v1/docs", nil))
	if docs.Code != http.StatusOK || !strings.Contains(docs.Body.String(), `"id":"a"`) {
		t.Fatalf("%d %s", docs.Code, docs.Body.String())
	}
}
