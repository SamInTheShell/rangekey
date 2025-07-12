package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/samintheshell/rangekey/internal/cli"
	cliv3 "github.com/urfave/cli/v3"
)

func main() {
	app := &cliv3.Command{
		Name:  "rangedb",
		Usage: "RangeDB - A distributed key-value database",
		Description: `RangeDB is a distributed key-value database built on etcd Raft with
automatic partitioning, transactions, and backup capabilities.

Run './rangedb --help' to see available commands.`,
		Commands: []*cliv3.Command{
			cli.NewServerCommand(),
			cli.NewGetCommand(),
			cli.NewPutCommand(),
			cli.NewDeleteCommand(),
			cli.NewRangeCommand(),
			cli.NewTxnCommand(),
			cli.NewBatchCommand(),
			cli.NewAdminCommand(),
			cli.NewVersionCommand(),
		},
		Action: func(ctx context.Context, c *cliv3.Command) error {
			// Print help if no command is provided
			return cliv3.ShowAppHelp(c)
		},
	}

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")
		cancel()
	}()

	if err := app.Run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}