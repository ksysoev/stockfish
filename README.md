# Stockfish Go

[![Tests](https://github.com/ksysoev/stockfish/actions/workflows/tests.yml/badge.svg)](https://github.com/ksysoev/stockfish/actions/workflows/tests.yml)
[![codecov](https://codecov.io/gh/ksysoev/stockfish/graph/badge.svg?token=CK8U257RQE)](https://codecov.io/gh/ksysoev/stockfish)
[![Go Report Card](https://goreportcard.com/badge/github.com/ksysoev/stockfish)](https://goreportcard.com/report/github.com/ksysoev/stockfish)
[![Go Reference](https://pkg.go.dev/badge/github.com/ksysoev/stockfish.svg)](https://pkg.go.dev/github.com/ksysoev/stockfish)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

A Go wrapper library for the Stockfish chess engine implementing the Universal Chess Interface (UCI) protocol. Provides a clean, idiomatic Go API for launching and communicating with the Stockfish process, setting options, configuring positions, streaming search results, and issuing both standard and non-standard UCI commands.

## Installation

```sh
go get github.com/ksysoev/stockfish@latest
```

## Usage

### Basic setup

Launch the engine, print its name and author, then close it cleanly.

```go
package main

import (
	"fmt"
	"log"

	"github.com/ksysoev/stockfish"
)

func main() {
	client, err := stockfish.New("/usr/local/bin/stockfish")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Println("Engine:", client.Name())
	fmt.Println("Author:", client.Author())
}
```

### Depth-limited search

Set the starting position, run a search to depth 15, and print the principal
variation at each depth along with the final best move.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/ksysoev/stockfish"
)

func main() {
	client, err := stockfish.New("/usr/local/bin/stockfish")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

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
}
```

### Configuring options

Pass one or more typed option constructors to `Apply`. Each constructor
validates the value against the engine's reported metadata (type, min/max
bounds, allowed combo values) before sending the command.

```go
package main

import (
	"log"

	"github.com/ksysoev/stockfish"
)

func main() {
	client, err := stockfish.New("/usr/local/bin/stockfish")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	if err = client.Apply(
		stockfish.WithThreads(4),
		stockfish.WithHash(256),
		stockfish.WithClearHash(),
	); err != nil {
		log.Fatal(err)
	}
}
```

Available named constructors:

| Constructor | UCI type | Option name |
|---|---|---|
| `WithThreads(n int)` | spin | `Threads` |
| `WithHash(mb int)` | spin | `Hash` |
| `WithPonder(v bool)` | check | `Ponder` |
| `WithMultiPV(n int)` | spin | `MultiPV` |
| `WithSkillLevel(n int)` | spin | `Skill Level` |
| `WithMoveOverhead(ms int)` | spin | `Move Overhead` |
| `WithClearHash()` | button | `Clear Hash` |
| `WithUCIChess960(v bool)` | check | `UCI_Chess960` |
| `WithSyzygyPath(path string)` | string | `SyzygyPath` |
| `WithUCIAnalyseMode(v bool)` | check | `UCI_AnalyseMode` |

For any other engine option use the generic constructors directly:

```go
client.Apply(
    stockfish.WithSpinOption("Minimum Thinking Time", 20),
    stockfish.WithComboOption("NumaPolicy", "auto"),
    stockfish.WithStringOption("Debug Log File", "/tmp/sf.log"),
    stockfish.WithButtonOption("Clear Hash"),
)
```

### Starting a new game

Call `NewGame` between games to reset the engine's transposition table and
search history so that previous analysis does not influence the new game.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ksysoev/stockfish"
)

func main() {
	client, err := stockfish.New("/usr/local/bin/stockfish")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	for _, moves := range [][]string{
		{"e2e4", "e7e5"},
		{"d2d4", "d7d5"},
	} {
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
}
```

## License

Stockfish Go is licensed under the MIT License. See the LICENSE file for more details.
