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

	fmt.Print(files)
	fmt.Print(unregisteredMsgs)

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

func findUnregisteredTypes(clientCtx client.Context, messages []map[string]any) (map[string]struct{}, error) {
	registry := clientCtx.Codec.InterfaceRegistry()
	unregisteredMsgs := make(map[string]struct{})
	for _, msg := range messages {
		for k, v := range msg {
			if k == "@type" {
				typeURL := v.(string)
				if _, err := registry.Resolve(typeURL); err != nil {
					// type not registered, add to the list if not already there
					unregisteredMsgs[typeURL] = struct{}{}
				}
				continue
			}
			switch x := v.(type) {
			case []map[string]any:
				for _, m := range x {
					tmpUnregisteredMsgs, err := findUnregisteredTypes(clientCtx, []map[string]any{m})
					if err != nil {
						return nil, err
					}
					for k := range tmpUnregisteredMsgs {
						unregisteredMsgs[k] = struct{}{}
					}
				}
			case map[string]any:
				tmpUnregisteredMsgs, err := findUnregisteredTypes(clientCtx, []map[string]any{x})
				if err != nil {
					return nil, err
				}
				for k := range tmpUnregisteredMsgs {
					unregisteredMsgs[k] = struct{}{}
				}
			}
		}
	}

	return unregisteredMsgs, nil
}
