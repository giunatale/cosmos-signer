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

// GetMultiSignCommand returns the transaction multi-sign command.
func GetMultiSignCommand() *cobra.Command {
	cmd := authcli.GetMultiSignCommand()
	cmd.Flags().String(flagPluginsDir, "", "The directory to search for plugin files")

	cmd.PreRun = preMultiSignCmd
	authMakeMultiSignCmd := cmd.RunE
	cmd.RunE = makeMultiSignCmd(authMakeMultiSignCmd)

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

func preMultiSignCmd(cmd *cobra.Command, _ []string) {
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

func makeMultiSignCmd(origMakeMultiSignCmd func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
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

		return origMakeMultiSignCmd(cmd, args)
	}
}

// GetMultiSignBatchCommand returns the transaction multi-sign-batch command.
func GetMultiSignBatchCommand() *cobra.Command {
	cmd := authcli.GetMultiSignBatchCmd()
	cmd.Flags().String(flagPluginsDir, "", "The directory to search for plugin files")

	cmd.PreRun = preMultiSignBatchCmd
	authMakeMultiSignBatchCmd := cmd.RunE
	cmd.RunE = makeMultiSignBatchCmd(authMakeMultiSignBatchCmd)

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

func preMultiSignBatchCmd(cmd *cobra.Command, _ []string) {
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

func makeMultiSignBatchCmd(origMakeMultiSignBatchCmd func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
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

		return origMakeMultiSignBatchCmd(cmd, args)
	}
}
