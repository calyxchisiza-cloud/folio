package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"folio"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:5320", "listen address")
	dir := flag.String("dir", "data", "index directory")
	flag.Parse()

	idx, err := folio.Open(*dir)
	if err != nil {
		fail(err)
	}
	defer idx.Close()
	srv, err := folio.Serve(*listen, idx)
	if err != nil {
		fail(err)
	}
	log.Printf("folio %s", *listen)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	<-ctx.Done()
	shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shut)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
