package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"cosmossdk.io/client/v2/autocli"
	clientv2keyring "cosmossdk.io/client/v2/autocli/keyring"
	"cosmossdk.io/core/address"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	txmodule "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	"github.com/cosmos/cosmos-sdk/x/auth/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/cosmos-signer/app"
)

const (
	flagBech32Prefix = "bech32-prefix"
	flagPrefixPublic = "prefix-pub"
)

// NewRootCmd creates a new root command for cosmos-signer. It is called once in the main function.
func NewRootCmd() *cobra.Command {
	var (
		txConfigOpts       tx.ConfigOptions
		autoCliOpts        autocli.AppOptions
		moduleBasicManager module.BasicManager
		clientCtx          client.Context
	)

	rootCmd := &cobra.Command{
		Use:           app.Name,
		Short:         "Run cosmossigner",
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			clientCtx = clientCtx.WithCmdContext(cmd.Context())

			bech32prefix, err := cmd.Flags().GetString(flagBech32Prefix)
			if err != nil {
				return err
			}
			if err := depinject.Inject(
				depinject.Configs(app.NewAppConfigWithBech32Prefix(bech32prefix),
					depinject.Supply(
						log.NewNopLogger(),
					),
					depinject.Provide(
						ProvideClientContext,
						// ProvideKeyring,
					),
				),
				&txConfigOpts,
				&autoCliOpts,
				&moduleBasicManager,
				&clientCtx,
			); err != nil {
				panic(err)
			}

			clientCtx, err = client.ReadPersistentCommandFlags(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			prefixPublic, err := cmd.Flags().GetString(flagPrefixPublic)
			if err != nil {
				return err
			}
			bech32PrefixAccPub := bech32prefix + prefixPublic
			// set the bech32 prefix
			sdkCfg := sdk.GetConfig()
			sdkCfg.SetBech32PrefixForAccount(bech32prefix, bech32PrefixAccPub)
			sdkCfg.Seal()

			clientCtx, err = config.ReadFromClientConfig(clientCtx)
			if err != nil {
				return err
			}

			// This needs to go after ReadFromClientConfig, as that function
			// sets the RPC client needed for SIGN_MODE_TEXTUAL.
			txConfigOpts.EnabledSignModes = append(txConfigOpts.EnabledSignModes, signing.SignMode_SIGN_MODE_TEXTUAL)
			txConfigOpts.TextualCoinMetadataQueryFn = txmodule.NewGRPCCoinMetadataQueryFn(clientCtx)
			txConfigWithTextual, err := tx.NewTxConfigWithOptions(
				codec.NewProtoCodec(clientCtx.InterfaceRegistry),
				txConfigOpts,
			)
			if err != nil {
				return err
			}

			clientCtx = clientCtx.WithTxConfig(txConfigWithTextual)
			if err := client.SetCmdClientContextHandler(clientCtx, cmd); err != nil {
				return err
			}

			if err := client.SetCmdClientContextHandler(clientCtx, cmd); err != nil {
				return err
			}

			return nil
		},
	}

	initRootCmd(rootCmd)
	rootCmd.PersistentFlags().String(flagBech32Prefix, sdk.Bech32MainPrefix, "The Bech32 prefix encoding for the signer address")
	rootCmd.PersistentFlags().String(flagPrefixPublic, sdk.PrefixPublic, "The prefix for public keys")

	overwriteFlagDefaults(rootCmd, map[string]string{
		flags.FlagChainID:        strings.ReplaceAll(app.Name, "-", ""),
		flags.FlagKeyringBackend: "test",
	})

	//if err := autoCliOpts.EnhanceRootCommand(rootCmd); err != nil {
	//	panic(err)
	//}

	return rootCmd
}

func overwriteFlagDefaults(c *cobra.Command, defaults map[string]string) {
	set := func(s *pflag.FlagSet, key, val string) {
		if f := s.Lookup(key); f != nil {
			f.DefValue = val
			_ = f.Value.Set(val)
		}
	}
	for key, val := range defaults {
		set(c.Flags(), key, val)
		set(c.PersistentFlags(), key, val)
	}
	for _, c := range c.Commands() {
		overwriteFlagDefaults(c, defaults)
	}
}

func ProvideClientContext(
	appCodec codec.Codec,
	interfaceRegistry codectypes.InterfaceRegistry,
	txConfig client.TxConfig,
	legacyAmino *codec.LegacyAmino,
) client.Context {
	clientCtx := client.Context{}.
		WithCodec(appCodec).
		WithInterfaceRegistry(interfaceRegistry).
		WithTxConfig(txConfig).
		WithLegacyAmino(legacyAmino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithHomeDir(app.DefaultNodeHome).
		WithViper(app.Name) // env variable prefix

	// Read the config again to overwrite the default values with the values from the config file
	clientCtx, _ = config.ReadFromClientConfig(clientCtx)

	return clientCtx
}

func ProvideKeyring(clientCtx client.Context, addressCodec address.Codec) (clientv2keyring.Keyring, error) {
	kb, err := client.NewKeyringFromBackend(clientCtx, clientCtx.Keyring.Backend())
	if err != nil {
		return nil, err
	}

	return keyring.NewAutoCLIKeyring(kb)
}
