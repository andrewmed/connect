package main

import (
	"context"
	"go.andmed.org/connect/lib"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	ctx := context.Background()

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("getwd: %w", err)
	}

	states, err := connect.GetStates()
	if err != nil {
		log.Fatalf("get states: %v", err)
	}

	state, err := connect.MatchingState(ctx, states, pwd)
	if err != nil {
		log.Fatalf("find session: %v", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		err = connect.ConnectState(ctx, state, pwd)
		if err != nil {
			log.Print(err)
		}
		cancel()
	}()

	select {
	case <-sig:
		cancel()
	case <-ctx.Done():
	}
}
