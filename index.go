package folio

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var stop = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "is": {}, "in": {}, "of": {}, "and": {}, "to": {}, "for": {},
}

// Index stores documents and the terms inside them.
type Index struct {
	mu    sync.Mutex
	dir   string
	file  *os.File
	docs  map[string]string
	post  map[string]map[string]struct{}
	alive bool
}

// New keeps the index in memory.
func New() *Index {
	return &Index{
		docs:  map[string]string{},
		post:  map[string]map[string]struct{}{},
		alive: true,
	}
}

// Open loads index.log from dir. A torn final line is dropped.
func Open(dir string) (*Index, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "index.log")
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	idx := New()
	idx.dir = dir
	kept := raw
	if len(raw) > 0 && raw[len(raw)-1] != '\n' {
		cut := bytes.LastIndexByte(raw, '\n')
		if cut < 0 {
			kept = nil
		} else {
			kept = raw[:cut+1]
		}
	}
	if err := idx.replay(kept); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := f.Truncate(int64(len(kept))); err != nil {
		f.Close()
		return nil, err
	}
	if _, err := f.Seek(int64(len(kept)), io.SeekStart); err != nil {
		f.Close()
		return nil, err
	}
	idx.file = f
	return idx, nil
}

// Close flushes the log.
func (idx *Index) Close() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.alive = false
	if idx.file == nil {
		return nil
	}
	err := idx.file.Close()
	idx.file = nil
	return err
}

// Add stores text under id, replacing a previous document with that id.
func (idx *Index) Add(id, text string) error {
	if err := validID(id); err != nil {
		return err
	}
	if len(text) > maxText {
		return ErrTooLarge
	}
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if !idx.alive {
		return ErrClosed
	}
	if err := idx.append(record{Op: "add", ID: id, Text: text}); err != nil {
		return err
	}
	idx.apply(record{Op: "add", ID: id, Text: text})
	return nil
}

// Delete removes a document.
func (idx *Index) Delete(id string) error {
	if err := validID(id); err != nil {
		return err
	}
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if !idx.alive {
		return ErrClosed
	}
	if _, ok := idx.docs[id]; !ok {
		return ErrNotFound
	}
	if err := idx.append(record{Op: "del", ID: id}); err != nil {
		return err
	}
	idx.apply(record{Op: "del", ID: id})
	return nil
}

// Search returns documents that contain any query term.
// Terms is how many distinct query terms the document contains.
// The highest count comes first. Equal counts are ordered by id.
func (idx *Index) Search(query string) ([]Hit, error) {
	terms := unique(tokens(query))
	if len(terms) == 0 {
		return nil, ErrInvalid
	}
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if !idx.alive {
		return nil, ErrClosed
	}
	score := map[string]int{}
	for _, term := range terms {
		for id := range idx.post[term] {
			score[id]++
		}
	}
	hits := make([]Hit, 0, len(score))
	for id, n := range score {
		hits = append(hits, Hit{ID: id, Terms: n})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Terms != hits[j].Terms {
			return hits[i].Terms > hits[j].Terms
		}
		return hits[i].ID < hits[j].ID
	})
	return hits, nil
}

func (idx *Index) replay(raw []byte) error {
	if len(raw) > 0 && raw[len(raw)-1] != '\n' {
		cut := bytes.LastIndexByte(raw, '\n')
		if cut < 0 {
			raw = nil
		} else {
			raw = raw[:cut+1]
		}
	}
	scan := bufio.NewScanner(bytes.NewReader(raw))
	scan.Buffer(make([]byte, 64*1024), maxText+1024)
	for scan.Scan() {
		var rec record
		if err := json.Unmarshal(scan.Bytes(), &rec); err != nil {
			return err
		}
		idx.apply(rec)
	}
	return scan.Err()
}

func (idx *Index) apply(rec record) {
	switch rec.Op {
	case "add":
		idx.clear(rec.ID)
		idx.docs[rec.ID] = rec.Text
		for _, term := range unique(tokens(rec.Text)) {
			if idx.post[term] == nil {
				idx.post[term] = map[string]struct{}{}
			}
			idx.post[term][rec.ID] = struct{}{}
		}
	case "del":
		idx.clear(rec.ID)
		delete(idx.docs, rec.ID)
	}
}

func (idx *Index) clear(id string) {
	text, ok := idx.docs[id]
	if !ok {
		return
	}
	for _, term := range unique(tokens(text)) {
		delete(idx.post[term], id)
		if len(idx.post[term]) == 0 {
			delete(idx.post, term)
		}
	}
}

func (idx *Index) append(rec record) error {
	if idx.file == nil {
		return nil
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if _, err := idx.file.Write(raw); err != nil {
		return err
	}
	return idx.file.Sync()
}

type record struct {
	Op   string `json:"op"`
	ID   string `json:"id"`
	Text string `json:"text,omitempty"`
}

func tokens(text string) []string {
	text = strings.ToLower(text)
	var cur []byte
	var out []string
	flush := func() {
		if len(cur) == 0 {
			return
		}
		word := string(cur)
		cur = cur[:0]
		if _, skip := stop[word]; skip {
			return
		}
		out = append(out, word)
	}
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			cur = append(cur, c)
			continue
		}
		flush()
	}
	flush()
	return out
}

func unique(words []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, word := range words {
		if _, ok := seen[word]; ok {
			continue
		}
		seen[word] = struct{}{}
		out = append(out, word)
	}
	return out
}

func validID(id string) error {
	if len(id) < 1 || len(id) > 64 {
		return ErrInvalid
	}
	for i, r := range id {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' || r == '-':
			if i == 0 {
				return ErrInvalid
			}
		default:
			return ErrInvalid
		}
	}
	return nil
}
