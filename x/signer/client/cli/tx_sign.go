package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcli "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
)

const (
	flagBech32Prefix = "bech32-prefix"
	flagPrefixPublic = "prefix-pub"
	flagPluginsDir   = "plugins-dir"
)

// GetSignCommand returns the transaction sign command.
func GetSignCommand() *cobra.Command {
	cmd := authcli.GetSignCommand()
	cmd.Flags().String(flagBech32Prefix, sdk.Bech32MainPrefix, "The Bech32 prefix encoding for the signer address")
	cmd.Flags().String(flagPrefixPublic, sdk.PrefixPublic, "The prefix for public keys")
	cmd.Flags().String(flagPluginsDir, "", "The directory to search for plugin files")

	cmd.PreRun = preSignCmd
	authMakeSignCmd := cmd.RunE
	cmd.RunE = makeSignCmd(authMakeSignCmd)

	return cmd
}

func preSignCmd(cmd *cobra.Command, _ []string) {
	err := cmd.MarkFlagRequired(flags.FlagOffline)
	if err != nil {
		panic(err)
	}

	err = cmd.MarkFlagRequired(flags.FlagAccountNumber)
	if err != nil {
		panic(err)
	}
	err = cmd.MarkFlagRequired(flags.FlagSequence)
	if err != nil {
		panic(err)
	}

	err = cmd.MarkFlagRequired(flagBech32Prefix)
	if err != nil {
		panic(err)
	}

	err = cmd.MarkFlagRequired(flagPluginsDir)
	if err != nil {
		panic(err)
	}
}

func makeSignCmd(origMakeSignCmd func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) (err error) {
		var clientCtx client.Context

		clientCtx, err = client.GetClientTxContext(cmd)
		if err != nil {
			return err
		}

		bech32prefix, err := cmd.Flags().GetString(flagBech32Prefix)
		if err != nil {
			return err
		}
		prefixPublic, err := cmd.Flags().GetString(flagPrefixPublic)
		if err != nil {
			return err
		}
		bech32PrefixAccPub := bech32prefix + prefixPublic
		// set the bech32 prefix
		config := sdk.GetConfig()
		config.SetBech32PrefixForAccount(bech32prefix, bech32PrefixAccPub)
		config.Seal()

		filename := args[0]
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close()
		var rawTx struct {
			Body struct {
				Messages []map[string]any
			}
		}
		if err := json.NewDecoder(f).Decode(&rawTx); err != nil {
			return fmt.Errorf("JSON decode %s: %v", filename, err)
		}
		unregisteredMsgs, err := findUnregisteredTypes(clientCtx, rawTx.Body.Messages)
		if err != nil {
			return err
		}

		fmt.Print(unregisteredMsgs)

		if len(unregisteredMsgs) > 0 {
			pluginsDir, err := cmd.Flags().GetString(flagPluginsDir)
			if err != nil {
				return err
			}

			err = RegisterSdkMsgsDynamic(clientCtx.Codec.InterfaceRegistry(), pluginsDir, unregisteredMsgs)
			if err != nil {
				return err
			}

			client.SetCmdClientContext(cmd, clientCtx) // not sure if this is needed
		}

		return origMakeSignCmd(cmd, args)
	}
}
