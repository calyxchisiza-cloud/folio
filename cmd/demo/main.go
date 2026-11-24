package main

import (
	"fmt"
	"os"

	"folio"
)

func main() {
	idx := folio.New()
	if err := idx.Add("a", "blue whale in the bay"); err != nil {
		fail(err)
	}
	if err := idx.Add("b", "the bay is quiet"); err != nil {
		fail(err)
	}
	hits, err := idx.Search("bay whale")
	if err != nil {
		fail(err)
	}
	for _, hit := range hits {
		fmt.Printf("%s %d\n", hit.ID, hit.Terms)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
