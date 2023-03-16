package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path/filepath"

	"golang.org/x/term"

	"github.com/ignition-pillar/go-zdk/client"
	signer "github.com/ignition-pillar/go-zdk/wallet"
	"github.com/ignition-pillar/go-zdk/zdk"
	"github.com/shopspring/decimal"
	"github.com/tyler-smith/go-bip39"
	"github.com/urfave/cli/v2"
	"github.com/zenon-network/go-zenon/wallet"
	//"github.com/faith/color"
	// TODO color
)

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

func getZnnCliSigner(walletDir string, cCtx *cli.Context) (signer.Signer, error) {

	var keyStorePath string

	// TODO use go-zdk keystore manager when available
	files, err := ioutil.ReadDir(walletDir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		fmt.Println("Error! No keystore in the default directory")
		return nil, nil
	} else if cCtx.IsSet("keyStore") {
		keyStorePath = filepath.Join(walletDir, cCtx.String("keyStore"))
		info, err := os.Stat(keyStorePath)
		if os.IsNotExist(err) || info.IsDir() {
			fmt.Println("Error! The keyStore", cCtx.String("keyStore"), "does not exist in the default directory")
			return nil, nil
		}
	} else if len(files) == 1 {
		fmt.Println("Using the default keyStore", files[0].Name())
		keyStorePath = filepath.Join(walletDir, files[0].Name())
	} else {
		fmt.Println("Error! Please provide a keyStore or an address. Use 'wallet.list' to list all available keyStores")
		return nil, nil
	}

	var passphrase string
	if !cCtx.IsSet("passphrase") {
		fmt.Println("Insert passphrase:")
		pw, err := term.ReadPassword(int(os.Stdin.Fd()))
		passphrase = string(pw)
		if err != nil {
			return nil, err
		}
	} else {
		passphrase = cCtx.String("passphrase")
	}

	kf, err := wallet.ReadKeyFile(keyStorePath)
	if err != nil {
		return nil, err
	}
	ks, err := kf.Decrypt(passphrase)
	if err != nil {
		if err == wallet.ErrWrongPassword {
			fmt.Println("Error! Invalid passphrase for keyStore", cCtx.String("keyStore"))
			return nil, nil
		}
		return nil, err
	}

	_, keyPair, err := ks.DeriveForIndexPath(uint32(cCtx.Int("index")))
	kp := signer.NewSigner(keyPair)

	return kp, nil

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
	walletDir := filepath.Join(nomctlDir, "wallet")
	err = os.MkdirAll(walletDir, os.FileMode(mode))
	if err != nil {
		log.Fatal(err)
	}

	var url string
	var chainId int

	znnCliBalance := &cli.Command{
		Name: "balance",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 0 {
				fmt.Println("Incorrect number of arguments. Expected:")
				fmt.Println("balance")
				return nil
			}
			kp, err := getZnnCliSigner(walletDir, cCtx)
			if err != nil {
				return err
			}
			if kp == nil {
				return nil
			}

			z, err := connect(url, chainId)
			if err != nil {
				return err
			}
			info, err := z.Ledger.GetAccountInfoByAddress(kp.Address())
			if err != nil {
				return err
			}
			fmt.Println("Balance for account-chain", kp.Address().String(), "having height", info.AccountHeight)
			if len(info.BalanceInfoMap) == 0 {
				fmt.Println("  No coins or tokens at address", kp.Address().String())
			}
			for zts, entry := range info.BalanceInfoMap {
				fmt.Println(" ", formatAmount(entry.Balance, entry.TokenInfo.Decimals), entry.TokenInfo.TokenSymbol, entry.TokenInfo.TokenDomain, zts.String())
			}
			return nil
		},
	}

	znnCliFrontierMomentum := &cli.Command{
		Name: "frontierMomentum",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 0 {
				fmt.Println("Incorrect number of arguments. Expected:")
				fmt.Println("frontierMomentum")
				return nil
			}
			z, err := connect(url, chainId)
			if err != nil {
				return err
			}
			m, err := z.Ledger.GetFrontierMomentum()
			if err != nil {
				return err
			}
			fmt.Println("Momentum height:", m.Height)
			fmt.Println("Momentum hash:", m.Hash.String())
			fmt.Println("Momentum previousHash:", m.PreviousHash.String())
			fmt.Println("Momentum timestamp:", m.TimestampUnix)
			return nil
		},
	}

	znnCliWalletCreateNew := &cli.Command{
		Name:  "wallet.createNew",
		Usage: "passphrase [keyStoreName]",
		Action: func(cCtx *cli.Context) error {
			if !(cCtx.NArg() == 1 || cCtx.NArg() == 2) {
				fmt.Println("Incorrect number of arguments. Expected:")
				fmt.Println("wallet.createNew passphrase [keyStoreName]")
				return nil
			}

			// TODO finally implement a local keystore manager in go-zdk?
			entropy, _ := bip39.NewEntropy(256)
			mnemonic, _ := bip39.NewMnemonic(entropy)
			ks := &wallet.KeyStore{
				Entropy:  entropy,
				Seed:     bip39.NewSeed(mnemonic, ""),
				Mnemonic: mnemonic,
			}
			_, kp, _ := ks.DeriveForIndexPath(0)
			ks.BaseAddress = kp.Address

			name := ks.BaseAddress.String()
			if cCtx.NArg() == 2 {
				name = cCtx.Args().Get(1)
			}

			password := cCtx.Args().Get(0)
			kf, _ := ks.Encrypt(password)
			kf.Path = filepath.Join(walletDir, name)
			//kf.Write()
			// Uncomment when file mode is fixed
			keyFileJson, err := json.MarshalIndent(kf, "", "    ")
			if err != nil {
				return err
			}
			os.WriteFile(kf.Path, keyFileJson, 0600)

			fmt.Println("keyStore successfully created:", name)
			return nil
		},
	}

	znnCliWalletList := &cli.Command{
		Name:  "wallet.list",
		Usage: "",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 0 {
				fmt.Println("Incorrect number of arguments. Expected:")
				fmt.Println("wallet.list")
				return nil
			}
			files, err := ioutil.ReadDir(walletDir)
			if err != nil {
				return err
			}
			if len(files) > 0 {
				fmt.Println("Available keyStores:")
				for _, f := range files {
					if !f.IsDir() {
						fmt.Println(f.Name())
					}
				}
			} else {
				fmt.Println("No keyStores found")
			}
			return nil
		},
	}
	
	znnCliPlasmaGet := &cli.Command{
		Name: "plasma.get",
		Usage: "",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 0 {
				fmt.Println("Incorrect number of arguments. Expected:")
				fmt.Println("plasma.get")
				return nil
			}

			kp, err := getZnnCliSigner(walletDir, cCtx)
			if err != nil{
				fmt.Println("Error getting signer:", err)
				return err
			}
			z, err := connect(url, chainId)
			if err != nil{
				fmt.Println("Error connecting to Zenon Network:", err)
				return err
			}
			plasmaInfo, err := z.Embedded.Plasma.Get(kp.Address())
			if err != nil {
				fmt.Println("Error getting plasma info:", err)
				return err
			}
			currentPlasma := plasmaInfo.CurrentPlasma
			maxPlasma := plasmaInfo.MaxPlasma
			qsrAmount := plasmaInfo.QsrAmount

			fmt.Printf("%s has %v/%v plasma with %v QSR fused.\n", kp.Address(), currentPlasma, maxPlasma, qsrAmount)
			return nil
		},
	}

	znnCliSubcommands := []*cli.Command{
		znnCliBalance,
		znnCliFrontierMomentum,
		znnCliWalletCreateNew,
		znnCliWalletList,
		znnCliPlasmaGet,
	}

	app := &cli.App{
		Name:  "nomctl",
		Usage: "A community controller for the Network of Momentum",
		Commands: []*cli.Command{
			{
				Name:        "znn-cli",
				Usage:       "A port of znn_cli_dart",
				Subcommands: znnCliSubcommands,
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
