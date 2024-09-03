package cli

import (
	"fmt"
	"path/filepath"
	"plugin"
	"strings"
	"unicode"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func RegisterTypes(ctx client.Context, pluginsDir string, unregisteredTypes map[string]struct{}) error {
	legacyAminoCodec := ctx.LegacyAmino
	registry := ctx.Codec.InterfaceRegistry()
	files, err := filepath.Glob(filepath.Join(pluginsDir, "*.so"))
	if err != nil {
		return err
	}

	lookupPaths := getLookupPackages(unregisteredTypes)

	for symbolName := range lookupPaths {
		symbolFound := false
		for _, file := range files {
			p, err := plugin.Open(file)
			if err != nil {
				return err
			}
			packageName := sanitizeSymbolName(symbolName)
			symRegisterLegacyAminoCodec := fmt.Sprintf("%s_RegisterLegacyAminoCodec", packageName)
			symRegisterLegacyAminoCodecObj, err := p.Lookup(symRegisterLegacyAminoCodec)
			if err != nil {
				continue
			}
			registerLegacyAminoCodec, ok := symRegisterLegacyAminoCodecObj.(*func(*codec.LegacyAmino))
			if !ok {
				return fmt.Errorf("failed to load %s", symRegisterLegacyAminoCodec)
			}
			(*registerLegacyAminoCodec)(legacyAminoCodec)

			symRegisterInterfaces := fmt.Sprintf("%s_RegisterInterfaces", packageName)
			symRegisterInterfacesObj, err := p.Lookup(symRegisterInterfaces)
			if err != nil {
				continue
			}
			registerInterfaces, ok := symRegisterInterfacesObj.(*func(cdctypes.InterfaceRegistry))
			if !ok {
				return fmt.Errorf("failed to load %s", symRegisterInterfaces)
			}
			(*registerInterfaces)(registry)
			symbolFound = true
		}
		if !symbolFound {
			return fmt.Errorf("failed to lookup symbol %s", symbolName)
		}
	}

	return nil
}

func findUnregisteredTypes(clientCtx client.Context, messages []map[string]any) (map[string]struct{}, error) {
	registry := clientCtx.Codec.InterfaceRegistry()
	unregisteredTypes := make(map[string]struct{})
	for _, msg := range messages {
		for k, v := range msg {
			if k == "@type" {
				typeURL := v.(string)
				if _, err := registry.Resolve(typeURL); err != nil {
					// type not registered, add to the list if not already there
					unregisteredTypes[typeURL] = struct{}{}
				}
				continue
			}
			switch x := v.(type) {
			case []map[string]any:
				for _, m := range x {
					tmpUnregisteredTypes, err := findUnregisteredTypes(clientCtx, []map[string]any{m})
					if err != nil {
						return nil, err
					}
					for k := range tmpUnregisteredTypes {
						unregisteredTypes[k] = struct{}{}
					}
				}
			case map[string]any:
				tmpUnregisteredTypes, err := findUnregisteredTypes(clientCtx, []map[string]any{x})
				if err != nil {
					return nil, err
				}
				for k := range tmpUnregisteredTypes {
					unregisteredTypes[k] = struct{}{}
				}
			}
		}
	}

	return unregisteredTypes, nil
}

// getLookupPackages outputs the list of package URLs
// for lookup in the plugins directory.
func getLookupPackages(unregisteredTypes map[string]struct{}) map[string]struct{} {
	// if type URL is `"/cosmos.bank.v1beta1.MsgSend"`, the package URL is `"/cosmos.bank.v1beta1"`
	// we assume the package URL is the prefix of the type URL up to the last `.`
	lookupPaths := make(map[string]struct{})
	for typeURL := range unregisteredTypes {
		s := strings.Split(typeURL, ".")
		s = s[:len(s)-1]
		lookupPath := strings.Join(s, ".")
		if _, ok := lookupPaths[lookupPath]; !ok {
			lookupPaths[lookupPath] = struct{}{}
		}
	}
	return lookupPaths
}

// sanitizeSymbolName sanitizes the type URL removing
// the leading `/`, replacing `.` with `_` and uppercasing
// the first character.
func sanitizeSymbolName(symbolName string) string {
	if symbolName == "" {
		return ""
	}

	symbolName = symbolName[1:]
	symbolName = strings.ReplaceAll(symbolName, ".", "_")
	symbolName = capitalizeFirstChar(symbolName)
	return symbolName
}

func capitalizeFirstChar(s string) string {
	if s == "" {
		return s
	}
	rs := []rune(s)
	rs[0] = unicode.ToUpper(rs[0])
	return string(rs)
}
