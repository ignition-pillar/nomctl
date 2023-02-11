package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ignition-pillar/go-zdk/client"
	"github.com/ignition-pillar/go-zdk/zdk"
	"github.com/urfave/cli/v2"
)

func connect(url string, chainId int) (*zdk.Zdk, error) {
	rpc, err := client.NewClient(url, client.ChainIdentifier(uint64(chainId)))
	if err != nil {
		return nil, err
	}
	z := zdk.NewZdk(rpc)
	return z, nil
}

func main() {

	home_dir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	nomctl_dir := filepath.Join(home_dir, ".nomctl")
	mode := int(0700)
	err = os.MkdirAll(nomctl_dir, os.FileMode(mode))
	if err != nil {
		log.Fatal(err)
	}
	wallet_dir := filepath.Join(nomctl_dir, "wallet")
	err = os.MkdirAll(wallet_dir, os.FileMode(mode))
	if err != nil {
		log.Fatal(err)
	}

	var url string
	var chainId int

	znn_cli_frontierMomentum := &cli.Command{
		Name: "frontierMomentum",
		Action: func(cCtx *cli.Context) error {
			z, err := connect(url, chainId)
			if err != nil {
				return err
			}
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
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "url",
						Aliases:     []string{"u"},
						Usage:       "Provide a websocket znnd connection URL with a port",
						Value:       "ws://127.0.0.1:35998",
						Destination: &url,
					},
					&cli.IntFlag{
						Name:        "chainId",
						Aliases:     []string{"n"},
						Usage:       "Specify the chain idendtifier to use",
						Value:       1,
						Destination: &chainId,
					},
					&cli.StringFlag{
						Name:    "passphrase",
						Aliases: []string{"p"},
						Usage:   "use this passphrase for the keyStore or enter it manually in a secure way",
					},
					&cli.StringFlag{
						Name:    "keyStore",
						Aliases: []string{"k"},
						Usage:   "Select the local keyStore",
						Value:   "available keyStore if only one is present",
					},
					&cli.IntFlag{
						Name:    "index",
						Aliases: []string{"i"},
						Usage:   "Address index",
						Value:   0,
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Prints detailed information about the action that it performs",
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
