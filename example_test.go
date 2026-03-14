package stockfish_test

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/ksysoev/stockfish"
)

// ExampleNew demonstrates launching the engine, reading its identity, and
// shutting it down cleanly.
func ExampleNew() {
	client, err := stockfish.New("/usr/local/bin/stockfish")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Engine:", client.Name())
	fmt.Println("Author:", client.Author())

	if err = client.Close(); err != nil {
		log.Fatal(err)
	}
}

// ExampleNew_withOptions demonstrates launching the engine and applying typed
// options in a single call so the client is fully configured before use.
func ExampleNew_withOptions() {
	client, err := stockfish.New(
		"/usr/local/bin/stockfish",
		stockfish.WithThreads(4),
		stockfish.WithHash(256),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Engine:", client.Name())

	if err = client.Close(); err != nil {
		log.Fatal(err)
	}
}

// ExampleClient_Go demonstrates setting the starting position, running a
// depth-limited search, and printing the principal variation at each depth
// together with the final best move.
func ExampleClient_Go() {
	client, err := stockfish.New("/usr/local/bin/stockfish")
	if err != nil {
		log.Fatal(err)
	}

	if err = client.SetPosition(stockfish.StartPosition()); err != nil {
		log.Fatal(err)
	}

	ch, err := client.Go(context.Background(), &stockfish.SearchParams{Depth: 15})
	if err != nil {
		log.Fatal(err)
	}

	for info := range ch {
		if info.IsBestMove {
			fmt.Println("Best move:", info.BestMove)
			break
		}

		fmt.Printf("depth=%-3d score=%d cp  pv=%s\n",
			info.Depth, info.Score.Value, strings.Join(info.PV, " "))
	}

	if err = client.Close(); err != nil {
		log.Fatal(err)
	}
}

// ExampleClient_Apply demonstrates configuring engine options before starting
// a search using the typed option constructors.
func ExampleClient_Apply() {
	client, err := stockfish.New("/usr/local/bin/stockfish")
	if err != nil {
		log.Fatal(err)
	}

	// Set thread count and hash table size, then clear the hash table.
	if err = client.Apply(
		stockfish.WithThreads(4),
		stockfish.WithHash(256),
		stockfish.WithClearHash(),
	); err != nil {
		log.Fatal(err)
	}

	if err = client.Close(); err != nil {
		log.Fatal(err)
	}
}

// ExampleClient_NewGame demonstrates resetting the engine between games to
// clear the transposition table and search history.
func ExampleClient_NewGame() {
	client, err := stockfish.New("/usr/local/bin/stockfish")
	if err != nil {
		log.Fatal(err)
	}

	games := [][]string{
		{"e2e4", "e7e5"},
		{"d2d4", "d7d5"},
	}

	for _, moves := range games {
		if err = client.NewGame(); err != nil {
			log.Fatal(err)
		}

		pos := stockfish.StartPosition().WithMoves(moves...)
		if err = client.SetPosition(pos); err != nil {
			log.Fatal(err)
		}

		ch, err := client.Go(context.Background(), &stockfish.SearchParams{Depth: 10})
		if err != nil {
			log.Fatal(err)
		}

		for info := range ch {
			if info.IsBestMove {
				fmt.Printf("moves=%v  best=%s\n", moves, info.BestMove)
				break
			}
		}
	}

	if err = client.Close(); err != nil {
		log.Fatal(err)
	}
}
