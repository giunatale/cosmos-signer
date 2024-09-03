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
		txCommand(rootCmd),
		keys.Commands(),
	)
}

func txCommand(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			err := rootCmd.PersistentPreRunE(cmd, []string{})
			if err != nil {
				return err
			}
			cmd.SetOut(signercli.NewFilterNullKeysJSON(cmd.OutOrStdout()))
			return nil
		},
	}

	cmd.AddCommand(
		signercli.GetSignCommand(),
	)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	cmd.AddCommand(
		signercli.GetValidateSignaturesCommand(),
	)
	cmd.AddCommand(
		signercli.GetMultiSignCommand(),
	)
	cmd.AddCommand(
		signercli.GetMultiSignBatchCommand(),
	)
	cmd.AddCommand(
		signercli.GetSignBatchCommand(),
	)

	return cmd
}
