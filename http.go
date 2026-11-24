package folio

import (
	_ "embed"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"time"
)

//go:embed index.html
var indexHTML []byte

// Handler serves documents, search, and the management page.
func Handler(idx *Index) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})
	mux.HandleFunc("GET /v1/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(idx.Documents())
	})
	mux.HandleFunc("PUT /v1/docs/{id}", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, maxText+1))
		if err != nil {
			http.Error(w, ErrInvalid.Error(), http.StatusBadRequest)
			return
		}
		if err := idx.Add(r.PathValue("id"), string(body)); err != nil {
			writeErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("DELETE /v1/docs/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := idx.Delete(r.PathValue("id")); err != nil {
			writeErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /v1/search", func(w http.ResponseWriter, r *http.Request) {
		hits, err := idx.Search(r.URL.Query().Get("q"))
		if err != nil {
			writeErr(w, err)
			return
		}
		type row struct {
			ID    string `json:"id"`
			Terms int    `json:"terms"`
		}
		out := make([]row, len(hits))
		for i, hit := range hits {
			out[i] = row{ID: hit.ID, Terms: hit.Terms}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
	return mux
}

func writeErr(w http.ResponseWriter, err error) {
	code := http.StatusBadRequest
	if err == ErrNotFound {
		code = http.StatusNotFound
	}
	if err == ErrTooLarge {
		code = http.StatusRequestEntityTooLarge
	}
	http.Error(w, err.Error(), code)
}

// Serve listens on a loopback address.
func Serve(addr string, idx *Index) (*http.Server, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, ErrInvalid
	}
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return nil, ErrInvalid
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           Handler(idx),
		ReadHeaderTimeout: 5 * time.Second,
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	go func() { _ = srv.Serve(ln) }()
	return srv, nil
}
