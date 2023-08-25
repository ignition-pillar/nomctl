package main

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ignition-pillar/go-zdk/client"
	"github.com/ignition-pillar/go-zdk/zdk"
	"github.com/shopspring/decimal"
	"github.com/urfave/cli/v2"
	"github.com/zenon-network/go-zenon/common/types"
	//"github.com/faith/color"
	// TODO color
)

var url string
var chainId int
var walletDir string

const ZnnDecimals = 8
const QsrDecimals = 8

const rpcMaxPageSize = 1024

func connect(url string, chainId int) (*zdk.Zdk, error) {
	rpc, err := client.NewClient(url, client.ChainIdentifier(uint64(chainId)))
	if err != nil {
		return nil, err
	}
	z := zdk.NewZdk(rpc)
	return z, nil
}

func formatAmount(amount *big.Int, decimals uint8) string {
	return decimal.NewFromBigInt(amount, int32(decimals)*-1).String()
}

func main() {

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	nomctlDir := filepath.Join(homeDir, ".nomctl")
	mode := int(0700)
	err = os.MkdirAll(nomctlDir, os.FileMode(mode))
	if err != nil {
		log.Fatal(err)
	}
	walletDir = filepath.Join(nomctlDir, "wallet")
	err = os.MkdirAll(walletDir, os.FileMode(mode))
	if err != nil {
		log.Fatal(err)
	}

	utilsValidateAddress := &cli.Command{
		Name:  "validate-address",
		Usage: "",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 1 {
				fmt.Println("Incorrect number of arguments. Expected:")
				fmt.Println("validate-address address")
				return nil
			}
			a := cCtx.Args().Get(0)
			address, err := types.ParseAddress(a)
			if err != nil {
				return err
			}
			fmt.Println(address, "is a valid address")
			return nil
		},
	}

	utilsSubcommands := []*cli.Command{
		utilsValidateAddress,
	}

	app := &cli.App{
		Name:  "nomctl",
		Usage: "A community controller for the Network of Momentum",
		Commands: []*cli.Command{
			&znnCliCommand,
			{
				Name:        "utils",
				Usage:       "A collection of helper utilities",
				Subcommands: utilsSubcommands,
			},
			&devnetCommand,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
