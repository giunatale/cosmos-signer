package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client"
	authcli "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
)

const (
	flagBech32Prefix = "bech32-prefix"
	flagPluginsDir   = "plugins-dir"
)

// GetSignCommand returns the transaction sign command.
func GetSignCommand() *cobra.Command {
	cmd := authcli.GetSignCommand()
	cmd.Flags().String(flagBech32Prefix, sdk.Bech32MainPrefix, "The Bech32 prefix encoding for the signer address")
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

		filename := args[0]
		stdTx, err := authclient.ReadTxFromFile(clientCtx, filename)
		if err != nil {
			return err
		}

		unregisteredMsgs, err := findUnregisteredMsgs(clientCtx, stdTx)
		if err != nil {
			return err
		}

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
