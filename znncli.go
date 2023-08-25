package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ignition-pillar/go-zdk/utils"
	"github.com/ignition-pillar/go-zdk/utils/template"
	signer "github.com/ignition-pillar/go-zdk/wallet"
	"github.com/tyler-smith/go-bip39"
	"github.com/urfave/cli/v2"
	"github.com/zenon-network/go-zenon/common/types"
	"github.com/zenon-network/go-zenon/vm/constants"
	"github.com/zenon-network/go-zenon/wallet"
	"golang.org/x/term"
)

func getZnnCliSigner(walletDir string, cCtx *cli.Context) (signer.Signer, error) {

	var keyStorePath string

	// TODO use go-zdk keystore manager when available
	files, err := os.ReadDir(walletDir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		fmt.Println("Error! No keystore in the default directory")
		os.Exit(1)

	} else if cCtx.IsSet("keyStore") {
		keyStorePath = filepath.Join(walletDir, cCtx.String("keyStore"))
		info, err := os.Stat(keyStorePath)
		if os.IsNotExist(err) || info.IsDir() {
			fmt.Println("Error! The keyStore", cCtx.String("keyStore"), "does not exist in the default directory")
			os.Exit(1)
		}
	} else if len(files) == 1 {
		fmt.Println("Using the default keyStore", files[0].Name())
		keyStorePath = filepath.Join(walletDir, files[0].Name())
	} else {
		fmt.Println("Error! Please provide a keyStore or an address. Use 'wallet.list' to list all available keyStores")
		os.Exit(1)
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
			os.Exit(1)
		}
		return nil, err
	}

	_, keyPair, err := ks.DeriveForIndexPath(uint32(cCtx.Int("index")))
	if err != nil {
		return nil, err
	}
	kp := signer.NewSigner(keyPair)

	return kp, nil

}

var znnCliUnreceived = &cli.Command{
	Name:  "unreceived",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("unreceived")
			return nil
		}

		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			fmt.Println("Error getting signer:", err)
			return err
		}
		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}

		unreceived, err := z.Ledger.GetUnreceivedBlocksByAddress(kp.Address(), 0, 5)
		if err != nil {
			fmt.Println("Error fetching unreceived txs:", err)
			return err
		}
		if len(unreceived.List) == 0 {
			fmt.Println("Nothing to receive")
			return nil
		} else {
			if unreceived.More {
				fmt.Println("You have more than", unreceived.Count, "transaction(s) to receive")
			} else {
				fmt.Println("You have", unreceived.Count, "transaction(s) to receive")
			}
		}
		fmt.Println("Showing the first", unreceived.Count)
		for _, block := range unreceived.List {
			fmt.Println("Unreceived", formatAmount(block.Amount, block.TokenInfo.Decimals), block.TokenInfo.TokenSymbol, "from", block.Address, "Use the hash", block.Hash, "to receive")
		}
		return nil
	},
}

var znnCliReceiveAll = &cli.Command{
	Name:  "receiveAll",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("receiveAll")
			return nil
		}

		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			fmt.Println("Error getting signer:", err)
			return err
		}
		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}

		unreceived, err := z.Ledger.GetUnreceivedBlocksByAddress(kp.Address(), 0, 5)
		if err != nil {
			fmt.Println("Error fetching unreceived txs:", err)
			return err
		}
		if len(unreceived.List) == 0 {
			fmt.Println("Nothing to receive")
			return nil
		} else {
			if unreceived.More {
				fmt.Println("You have more than", unreceived.Count, "transaction(s) to receive")
			} else {
				fmt.Println("You have", unreceived.Count, "transaction(s) to receive")
			}
		}
		fmt.Println("Please wait ...")

		for unreceived.Count > 0 {
			for _, block := range unreceived.List {
				temp := template.Receive(1, uint64(chainId), block.Hash)
				_, err = utils.Send(z, temp, kp, false)
			}
			unreceived, err = z.Ledger.GetUnreceivedBlocksByAddress(kp.Address(), 0, 5)
			if err != nil {
				fmt.Println("Error fetching unreceived txs:", err)
				return err
			}
		}

		fmt.Println("Done")
		return nil
	},
}

var znnCliBalance = &cli.Command{
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

var znnCliFrontierMomentum = &cli.Command{
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

var znnCliWalletCreateNew = &cli.Command{
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

var znnCliWalletCreateFromMnemonic = &cli.Command{
	Name:  "wallet.createFromMnemonic",
	Usage: "passphrase [keyStoreName]",
	Action: func(cCtx *cli.Context) error {
		if !(cCtx.NArg() == 2 || cCtx.NArg() == 3) {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("wallet.createFromMnemonic \"mnemonic\" passphrase [keyStoreName]")
			return nil
		}

		// TODO finally implement a local keystore manager in go-zdk?
		ms := cCtx.Args().Get(0)
		// TODO add in validation
		entropy, _ := bip39.EntropyFromMnemonic(ms)
		mnemonic, _ := bip39.NewMnemonic(entropy)
		ks := &wallet.KeyStore{
			Entropy:  entropy,
			Seed:     bip39.NewSeed(mnemonic, ""),
			Mnemonic: mnemonic,
		}
		_, kp, _ := ks.DeriveForIndexPath(0)
		ks.BaseAddress = kp.Address

		name := ks.BaseAddress.String()
		if cCtx.NArg() == 3 {
			name = cCtx.Args().Get(2)
		}

		password := cCtx.Args().Get(1)
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

var znnCliWalletList = &cli.Command{
	Name:  "wallet.list",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("wallet.list")
			return nil
		}
		files, err := os.ReadDir(walletDir)
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

var znnCliPillarList = &cli.Command{
	Name:  "pillar.list",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("pillar.list")
			return nil
		}

		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}
		pillarInfoList, err := z.Embedded.Pillar.GetAll(0, rpcMaxPageSize)
		if err != nil {
			fmt.Println("Error getting pillar list:", err)
			return err
		}

		for _, p := range pillarInfoList.List {
			fmt.Printf("#%d Pillar %s has a delegated weight of %s ZNN\n", p.Rank+1, p.Name, formatAmount(p.Weight, ZnnDecimals))
			fmt.Printf("    Producer address %s\n", p.BlockProducingAddress)
			fmt.Printf("    Momentums %d / %d\n", p.CurrentStats.ProducedMomentums, p.CurrentStats.ExpectedMomentums)
		}
		return nil
	},
}

var znnCliPillarUncollected = &cli.Command{
	Name:  "pillar.uncollected",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("pillar.uncollected")
			return nil
		}

		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			fmt.Println("Error getting signer:", err)
			return err
		}
		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}
		uncollected, err := z.Embedded.Pillar.GetUncollectedReward(kp.Address())
		if err != nil {
			fmt.Println("Error getting uncollected pillar reward(s):", err)
			return err
		}
		if uncollected.Znn.Sign() != 0 {
			fmt.Println(formatAmount(uncollected.Znn, ZnnDecimals), "ZNN")
		}
		if uncollected.Qsr.Sign() != 0 {
			fmt.Println(formatAmount(uncollected.Qsr, ZnnDecimals), "QSR")
		}
		if uncollected.Znn.Sign() == 0 && uncollected.Qsr.Sign() == 0 {
			fmt.Println("No rewards to collect")
		}

		return nil
	},
}

var znnCliPillarCollect = &cli.Command{
	Name:  "pillar.collect",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("pillar.collect")
			return nil
		}

		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			fmt.Println("Error getting signer:", err)
			return err
		}
		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}
		template, err := z.Embedded.Pillar.CollectReward()
		if err != nil {
			fmt.Println("Error templating pillar collect tx:", err)
			return err
		}
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending pillar collect tx:", err)
			return err
		}

		fmt.Println("Done")
		fmt.Println("Use 'receiveAll' to collect your Pillar reward(s) after 1 momentum")
		return nil
	},
}

var znnCliPillarDelegate = &cli.Command{
	Name:  "pillar.delegate",
	Usage: "name",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 1 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("pillar.delegate name")
			return nil
		}

		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			fmt.Println("Error getting signer:", err)
			return err
		}
		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}

		pillar := cCtx.Args().Get(0)

		template, err := z.Embedded.Pillar.Delegate(pillar)
		if err != nil {
			fmt.Println("Error templating pillar delegate tx:", err)
			return err
		}
		fmt.Println("Delegating to Pillar", pillar)
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending pillar delegate tx:", err)
			return err
		}

		fmt.Println("Done")
		return nil
	},
}

var znnCliPillarUndelegate = &cli.Command{
	Name:  "pillar.undelegate",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("pillar.undelegate")
			return nil
		}

		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			fmt.Println("Error getting signer:", err)
			return err
		}
		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}
		template, err := z.Embedded.Pillar.Undelegate()
		if err != nil {
			fmt.Println("Error templating pillar undelegate tx:", err)
			return err
		}
		fmt.Println("Undelegating ...")
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending pillar undelegate tx:", err)
			return err
		}

		fmt.Println("Done")
		return nil
	},
}

var znnCliPlasmaGet = &cli.Command{
	Name:  "plasma.get",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("plasma.get")
			return nil
		}

		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			fmt.Println("Error getting signer:", err)
			return err
		}
		z, err := connect(url, chainId)
		if err != nil {
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
		formattedQsrAmount := formatAmount(plasmaInfo.QsrAmount, QsrDecimals)

		fmt.Printf("%s has %v/%v plasma with %v QSR fused.\n", kp.Address(), currentPlasma, maxPlasma, formattedQsrAmount)
		return nil
	},
}

var znnCliSporkList = &cli.Command{
	Name:  "spork.list",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("spork.list")
			return nil
		}

		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}
		sporkList, err := z.Embedded.Spork.GetAll(0, rpcMaxPageSize)
		if err != nil {
			fmt.Println("Error getting spork list:", err)
			return err
		}

		if len(sporkList.List) == 0 {
			fmt.Println("No sporks found")
		} else {
			fmt.Println("Sporks:")
			for _, s := range sporkList.List {
				fmt.Printf("Name: %v\n", s.Name)
				fmt.Printf("  Description: %v\n", s.Description)
				fmt.Printf("  Activated: %v\n", s.Activated)
				if s.Activated {
					fmt.Printf("  EnforcementHeight: %v\n", s.EnforcementHeight)
				}
				fmt.Printf("  Hash: %v\n", s.Id)
			}
		}

		return nil
	},
}

var znnCliSporkCreate = &cli.Command{
	Name:  "spork.create",
	Usage: "name description",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 2 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("spork.list name description")
			return nil
		}

		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			fmt.Println("Error getting signer:", err)
			return err
		}

		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}

		name := cCtx.Args().Get(0)
		if len(name) < constants.SporkNameMinLength || len(name) > constants.SporkNameMaxLength {
			fmt.Println("Spork name must be", constants.SporkNameMinLength, "to", constants.SporkNameMaxLength, "characters in length")
			return nil
		}
		description := cCtx.Args().Get(1)
		if len(description) > constants.SporkDescriptionMaxLength {
			fmt.Println("Spork description cannot exceed", constants.SporkDescriptionMaxLength, "characters in length")
		}

		template, err := z.Embedded.Spork.Create(name, description)
		if err != nil {
			fmt.Println("Error templating spork create tx:", err)
			return err
		}
		fmt.Println("Creating spork...")
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending spork create tx:", err)
			return err
		}

		fmt.Println("Done")
		return nil
	},
}

var znnCliSporkActivate = &cli.Command{
	Name:  "spork.activate",
	Usage: "id",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 1 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("spork.activate id")
			return nil
		}

		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			fmt.Println("Error getting signer:", err)
			return err
		}

		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}

		id := types.HexToHashPanic(cCtx.Args().Get(0))

		template, err := z.Embedded.Spork.Activate(id)
		if err != nil {
			fmt.Println("Error templating spork activate tx:", err)
			return err
		}
		fmt.Println("Activating spork...")
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending spork activate tx:", err)
			return err
		}

		fmt.Println("Done")
		return nil
	},
}
var znnCliSentinelUncollected = &cli.Command{
	Name:  "sentinel.uncollected",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("sentinel.uncollected")
			return nil
		}

		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			fmt.Println("Error getting signer:", err)
			return err
		}
		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}
		uncollected, err := z.Embedded.Sentinel.GetUncollectedReward(kp.Address())
		if err != nil {
			fmt.Println("Error getting uncollected sentinel reward(s):", err)
			return err
		}
		if uncollected.Znn.Sign() != 0 {
			fmt.Println(formatAmount(uncollected.Znn, ZnnDecimals), "ZNN")
		}
		if uncollected.Qsr.Sign() != 0 {
			fmt.Println(formatAmount(uncollected.Qsr, ZnnDecimals), "QSR")
		}
		if uncollected.Znn.Sign() == 0 && uncollected.Qsr.Sign() == 0 {
			fmt.Println("No rewards to collect")
		}

		return nil
	},
}

var znnCliSentinelCollect = &cli.Command{
	Name:  "sentinel.collect",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("sentinel.collect")
			return nil
		}

		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			fmt.Println("Error getting signer:", err)
			return err
		}
		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}
		template, err := z.Embedded.Sentinel.CollectReward()
		if err != nil {
			fmt.Println("Error templating sentinel collect tx:", err)
			return err
		}
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending sentinel collect tx:", err)
			return err
		}

		fmt.Println("Done")
		fmt.Println("Use 'receiveAll' to collect your Sentinel reward(s) after 1 momentum")
		return nil
	},
}

var znnCliStakeUncollected = &cli.Command{
	Name:  "stake.uncollected",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("stake.uncollected")
			return nil
		}

		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			fmt.Println("Error getting signer:", err)
			return err
		}
		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}
		uncollected, err := z.Embedded.Stake.GetUncollectedReward(kp.Address())
		if err != nil {
			fmt.Println("Error getting uncollected stake reward(s):", err)
			return err
		}
		if uncollected.Znn.Sign() != 0 {
			fmt.Println(formatAmount(uncollected.Znn, ZnnDecimals), "ZNN")
		}
		if uncollected.Qsr.Sign() != 0 {
			fmt.Println(formatAmount(uncollected.Qsr, ZnnDecimals), "QSR")
		}
		if uncollected.Znn.Sign() == 0 && uncollected.Qsr.Sign() == 0 {
			fmt.Println("No rewards to collect")
		}

		return nil
	},
}

var znnCliStakeCollect = &cli.Command{
	Name:  "stake.collect",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("stake.collect")
			return nil
		}

		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			fmt.Println("Error getting signer:", err)
			return err
		}
		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}
		template, err := z.Embedded.Stake.CollectReward()
		if err != nil {
			fmt.Println("Error templating stake collect tx:", err)
			return err
		}
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending stake collect tx:", err)
			return err
		}

		fmt.Println("Done")
		fmt.Println("Use 'receiveAll' to collect your stake reward(s) after 1 momentum")
		return nil
	},
}

var znnCliSubcommands = []*cli.Command{
	znnCliBalance,
	znnCliFrontierMomentum,
	znnCliWalletCreateNew,
	znnCliWalletCreateFromMnemonic,
	znnCliWalletList,
	//		znnCliWalletDeriveAddresses,
	znnCliPlasmaGet,
	znnCliPillarList,
	znnCliPillarUncollected,
	znnCliPillarCollect,
	znnCliPillarDelegate,
	znnCliPillarUndelegate,
	znnCliSporkList,
	znnCliSporkCreate,
	znnCliSporkActivate,
	znnCliSentinelUncollected,
	znnCliSentinelCollect,
	znnCliStakeUncollected,
	znnCliStakeCollect,
	znnCliReceiveAll,
	znnCliUnreceived,
}

var znnCliCommand = cli.Command{
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
}
