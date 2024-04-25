package cmd

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/spf13/cobra"

	signercli "github.com/atomone-hub/cosmos-signer/x/signer/client/cli"
)

func initRootCmd(
	rootCmd *cobra.Command,
) {
	// add keybase, auxiliary RPC, query, genesis, and tx child commands
	rootCmd.AddCommand(
		txCommand(),
		keys.Commands(),
	)
}

func txCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		signercli.GetSignCommand(),
	)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}
