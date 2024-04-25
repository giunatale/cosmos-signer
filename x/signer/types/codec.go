package types

import (
	"bufio"
	"os"
	"path/filepath"
	"plugin"
	"reflect"
	"strings"

	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	RegisterDynamicMsgs(registry, "/path/to/your/plugins")
}

func RegisterDynamicMsgs(registry cdctypes.InterfaceRegistry, pluginsDir string) error {
	files, err := filepath.Glob(filepath.Join(pluginsDir, "*.so"))
	if err != nil {
		return err
	}

	for _, file := range files {
		// the manifest file is assumed to be in the same place as the .so file
		// with a .manifest extension
		manifestFile := strings.TrimSuffix(file, ".so") + ".manifest"
		symbols, err := readManifest(manifestFile)
		if err != nil {
			return err
		}

		p, err := plugin.Open(file)
		if err != nil {
			return err
		}

		for _, symbolName := range symbols {
			sym, err := p.Lookup(symbolName)
			if err != nil {
				continue // Skip if symbol not found
			}

			msgType, ok := sym.(reflect.Type)
			if !ok || !reflect.PointerTo(msgType).Implements(reflect.TypeOf((*sdk.Msg)(nil)).Elem()) {
				continue // Skip if loaded type does not implement sdk.Msg
			}

			instance := reflect.New(msgType).Interface()
			if protoMsg, ok := instance.(sdk.Msg); ok {
				registry.RegisterImplementations((*sdk.Msg)(nil), protoMsg)
			}
		}
	}
	return nil
}

func readManifest(manifestFile string) ([]string, error) {
	var symbols []string
	file, err := os.Open(manifestFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		symbols = append(symbols, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return symbols, nil
}
