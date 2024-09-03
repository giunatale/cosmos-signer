package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	authcli "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
)

// GetSignCommand returns the transaction sign command.
func GetSignCommand() *cobra.Command {
	cmd := authcli.GetSignCommand()
	cmd.Flags().String(flagPluginsDir, "", "The directory to search for plugin files")

	cmd.PreRun = preSignCmd
	authMakeSignCmd := cmd.RunE
	cmd.RunE = makeSignCmd(authMakeSignCmd)

	cmd.PostRunE = func(cmd *cobra.Command, args []string) error {
		outputDoc, err := cmd.Flags().GetString(flags.FlagOutputDocument)
		if err != nil {
			return err
		}
		FilterNullJSONKeysFile(outputDoc)
		return nil
	}

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
		unregisteredTypes, err := findUnregisteredTypes(clientCtx, rawTx.Body.Messages)
		if err != nil {
			return err
		}

		if len(unregisteredTypes) > 0 {
			pluginsDir, err := cmd.Flags().GetString(flagPluginsDir)
			if err != nil {
				return err
			}
			err = RegisterTypes(clientCtx, pluginsDir, unregisteredTypes)
			if err != nil {
				return err
			}
		}

		return origMakeSignCmd(cmd, args)
	}
}
