package app

import (
	runtimev1alpha1 "cosmossdk.io/api/cosmos/app/runtime/v1alpha1"
	appv1alpha1 "cosmossdk.io/api/cosmos/app/v1alpha1"
	authmodulev1 "cosmossdk.io/api/cosmos/auth/module/v1"
	txconfigv1 "cosmossdk.io/api/cosmos/tx/config/v1"
	"cosmossdk.io/core/appconfig"
	"cosmossdk.io/depinject"
	_ "github.com/atomone-hub/cosmos-signer/x/signer" // import for side-effects
	"github.com/cosmos/cosmos-sdk/runtime"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	// this line is used by starport scaffolding # stargate/app/moduleImport
)

var (
	// appConfig application configuration (used by depinject)
	appConfig = NewAppConfigWithBech32Prefix(AccountAddressPrefix)
)

// NewAppConfigWithBech32Prefix creates a new app configuration with the provided Bech32 prefixes.
func NewAppConfigWithBech32Prefix(bech32Prefix string) depinject.Config {
	return appconfig.Compose(&appv1alpha1.Config{
		Modules: []*appv1alpha1.ModuleConfig{
			{
				Name: runtime.ModuleName,
				Config: appconfig.WrapAny(&runtimev1alpha1.Module{
					AppName: Name,
				}),
			},
			{
				Name: authtypes.ModuleName,
				Config: appconfig.WrapAny(&authmodulev1.Module{
					Bech32Prefix: bech32Prefix,
				}),
			},
			{
				Name:   "tx",
				Config: appconfig.WrapAny(&txconfigv1.Config{}),
			},
		},
	})
}
