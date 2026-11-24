// Package folio is a small full-text index.
// A search returns documents ordered by how many query terms they contain.
package folio

import "errors"

const maxText = 64 << 10

var (
	ErrInvalid  = errors.New("name is not valid")
	ErrNotFound = errors.New("document not found")
	ErrTooLarge = errors.New("document is too large")
	ErrClosed   = errors.New("index is closed")
)

// Hit is one document that contained at least one query term.
type Hit struct {
	ID    string
	Terms int
}
