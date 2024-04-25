package cli

import (
	"fmt"
	"path/filepath"
	"plugin"
	"reflect"

	"github.com/cosmos/cosmos-sdk/client"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterSdkMsgsDynamic(registry cdctypes.InterfaceRegistry, pluginsDir string, unregisteredMsgs map[string]struct{}) error {
	files, err := filepath.Glob(filepath.Join(pluginsDir, "*.so"))
	if err != nil {
		return err
	}

	for _, file := range files {
		p, err := plugin.Open(file)
		if err != nil {
			return err
		}

		for symbolName := range unregisteredMsgs {
			sym, err := p.Lookup(symbolName)
			if err != nil {
				return fmt.Errorf("failed to lookup symbol %s in %s: %w", symbolName, file, err)
			}

			msgType, ok := sym.(reflect.Type)
			if !ok || !reflect.PointerTo(msgType).Implements(reflect.TypeOf((*sdk.Msg)(nil)).Elem()) {
				return fmt.Errorf("symbol %s in %s is not a sdk.Msg", symbolName, file)
			}

			instance := reflect.New(msgType).Interface()
			if protoMsg, ok := instance.(sdk.Msg); ok {
				registry.RegisterImplementations((*sdk.Msg)(nil), protoMsg)
			}
		}
	}
	return nil
}

func findUnregisteredMsgs(clientCtx client.Context, stdTx sdk.Tx) (map[string]struct{}, error) {
	registry := clientCtx.Codec.InterfaceRegistry()
	unregisteredMsgs := make(map[string]struct{})
	for _, msg := range stdTx.GetMsgs() {
		anyMsg, err := cdctypes.NewAnyWithValue(msg)
		if err != nil {
			return nil, err
		}
		typeURL := anyMsg.TypeUrl
		if _, err := registry.Resolve(typeURL); err != nil {
			// type not registered, add to the list if not already there
			unregisteredMsgs[typeURL] = struct{}{}
		}
	}
	return unregisteredMsgs, nil
}
