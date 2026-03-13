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

// ExampleClient_SetOption demonstrates configuring engine options before
// starting a search. Spin options (like Threads) require a string value;
// button options (like Clear Hash) require nil.
func ExampleClient_SetOption() {
	client, err := stockfish.New("/usr/local/bin/stockfish")
	if err != nil {
		log.Fatal(err)
	}

	// Set thread count (spin option).
	threads := "4"
	if err = client.SetOption("Threads", &threads); err != nil {
		log.Fatal(err)
	}

	// Trigger a button option — no value needed.
	if err = client.SetOption("Clear Hash", nil); err != nil {
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
