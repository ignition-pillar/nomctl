package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ignition-pillar/go-zdk/client"
	"github.com/ignition-pillar/go-zdk/zdk"
	"github.com/urfave/cli/v2"
)

func main() {
	url := "ws://127.0.0.1:35998"
	rpc, err := client.NewClient(url)
	if err != nil {
		log.Fatal(err)
	}
	z := zdk.NewZdk(rpc)

	znn_cli_frontierMomentum := &cli.Command{
		Name: "frontierMomentum",
		Action: func(cCtx *cli.Context) error {
			m, err := z.Ledger.GetFrontierMomentum()
			if err != nil {
				return err
			}
			fmt.Printf("Momentum height: %d\n", m.Height)
			fmt.Printf("Momentum hash: %s\n", m.Hash.String())
			fmt.Printf("Momentum previousHash: %s\n", m.PreviousHash.String())
			fmt.Printf("Momentum timestamp: %d\n", m.TimestampUnix)
			return nil
		},
	}

	znn_cli_subcommands := []*cli.Command{
		znn_cli_frontierMomentum,
	}

	app := &cli.App{
		Name:  "nomctl",
		Usage: "A community controller for the Network of Momentum",
		Commands: []*cli.Command{
			{
				Name:        "znn-cli",
				Usage:       "A port of znn_cli_dart",
				Subcommands: znn_cli_subcommands,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
