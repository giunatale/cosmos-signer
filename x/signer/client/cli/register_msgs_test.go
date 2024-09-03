package cli

import (
	"errors"
	"fmt"
	"plugin"
	"reflect"
	"testing"

	"github.com/atomone-hub/cosmos-signer/x/signer/client/cli/testfiles"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	types "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

// MockPlugin defines a mock plugin for testing purposes
type MockPlugin struct {
	lookupResults map[string]interface{}
}

func (m *MockPlugin) Lookup(symName string) (plugin.Symbol, error) {
	if result, ok := m.lookupResults[symName]; ok {
		return result, nil
	}
	return nil, fmt.Errorf("symbol %s not found", symName)
}

func TestSanitizeSymbolName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty string", "", ""},
		{"Simple type URL", "/cosmos.bank.v1beta1.MsgSend", "Cosmos_bank_v1beta1_MsgSend"},
		{"Type URL with multiple dots", "/cosmos.bank.v1.beta1.MsgSend", "Cosmos_bank_v1_beta1_MsgSend"},
		{"Type URL with uppercase letters", "/cosmos.Bank.v1beta1.MsgSend", "Cosmos_Bank_v1beta1_MsgSend"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeSymbolName(tc.input)
			if result != tc.expected {
				t.Errorf("Expected: %s, Got: %s", tc.expected, result)
			}
		})
	}
}

func TestCapitalizeFirstChar(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty string", "", ""},
		{"Single lowercase letter", "a", "A"},
		{"Single uppercase letter", "A", "A"},
		{"Multiple letters", "hello", "Hello"},
		{"All uppercase letters", "WORLD", "WORLD"},
		{"Mixed case", "hEllO", "HEllO"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := capitalizeFirstChar(tc.input)
			if result != tc.expected {
				t.Errorf("Expected: %s, Got: %s", tc.expected, result)
			}
		})
	}
}

func TestGetLookupPackages(t *testing.T) {
	testCases := []struct {
		name          string
		unregistered  map[string]struct{}
		expectedPaths map[string]struct{}
	}{
		{
			name:          "Empty unregistered types",
			unregistered:  map[string]struct{}{},
			expectedPaths: map[string]struct{}{},
		},
		{
			name: "Single type URL",
			unregistered: map[string]struct{}{
				"/cosmos.bank.v1beta1.MsgSend": {},
			},
			expectedPaths: map[string]struct{}{
				"/cosmos.bank.v1beta1": {},
			},
		},
		{
			name: "Multiple type URLs with same prefix",
			unregistered: map[string]struct{}{
				"/cosmos.bank.v1beta1.MsgSend":    {},
				"/cosmos.bank.v1beta1.MsgDeposit": {},
			},
			expectedPaths: map[string]struct{}{
				"/cosmos.bank.v1beta1": {},
			},
		},
		{
			name: "Multiple type URLs with different prefixes",
			unregistered: map[string]struct{}{
				"/cosmos.bank.v1beta1.MsgSend":        {},
				"/cosmos.staking.v1beta1.MsgDelegate": {},
			},
			expectedPaths: map[string]struct{}{
				"/cosmos.bank.v1beta1":    {},
				"/cosmos.staking.v1beta1": {},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getLookupPackages(tc.unregistered)
			if !reflect.DeepEqual(result, tc.expectedPaths) {
				t.Errorf("Expected lookup paths: %v, Got: %v", tc.expectedPaths, result)
			}
		})
	}
}

func TestFindUnregisteredTypes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock InterfaceRegistry that implements the types.InterfaceRegistry interface
	mIRegistry := testfiles.NewMockInterfaceRegistry(ctrl)

	// Setup the expected method calls and return values for the mock InterfaceRegistry
	mIRegistry.EXPECT().Resolve("/cosmos.bank.v1beta1.MsgSend").Return(nil, nil).AnyTimes()                               // Registered type
	mIRegistry.EXPECT().Resolve("/cosmos.bank.v1beta1.MsgBurn").Return(nil, errors.New("type not registered")).AnyTimes() // Unregistered type

	// Create a mock Codec
	mCodec := testfiles.NewMockCodec(ctrl)

	// Setup the expected method calls and return values for the mock Codec
	mCodec.EXPECT().InterfaceRegistry().Return(mIRegistry).AnyTimes()

	clientCtx := client.Context{
		Codec: mCodec,
	}

	tests := []struct {
		name           string
		messages       []map[string]any
		expectedResult map[string]struct{}
		expectedError  bool
	}{
		{
			name: "No unregistered types",
			messages: []map[string]any{
				{"@type": "/cosmos.bank.v1beta1.MsgSend"},
			},
			expectedResult: map[string]struct{}{}, // No unregistered types expected
			expectedError:  false,
		},
		{
			name: "One unregistered type",
			messages: []map[string]any{
				{"@type": "/cosmos.bank.v1beta1.MsgBurn"},
			},
			expectedResult: map[string]struct{}{
				"/cosmos.bank.v1beta1.MsgBurn": {}, // Unregistered type expected
			},
			expectedError: false,
		},
		{
			name: "Nested messages with unregistered types",
			messages: []map[string]any{
				{
					"msg": []map[string]any{
						{"@type": "/cosmos.bank.v1beta1.MsgBurn"},
						{"@type": "/cosmos.bank.v1beta1.MsgSend"},
					},
				},
			},
			expectedResult: map[string]struct{}{
				"/cosmos.bank.v1beta1.MsgBurn": {}, // Only the unregistered type expected
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := findUnregisteredTypes(clientCtx, tt.messages)
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expectedResult, result)
		})
	}
}

// TestRegisterTypes defines tests for the RegisterTypes function
func TestRegisterTypes(t *testing.T) {
	type testCase struct {
		name         string
		ctxMock      *client.Context
		pluginsDir   string
		unregistered map[string]struct{}
		mockFiles    []string // Mocked list of plugin files
		mockLegacy   func(*testfiles.MockCodec) error
		mockRegistry func(*testfiles.MockInterfaceRegistry) error
		expectedErr  error
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock Codec
	mCodec := testfiles.NewMockCodec(ctrl)

	clientCtx := client.Context{
		Codec:       mCodec,
		LegacyAmino: codec.NewLegacyAmino(),
	}

	// Create a mock InterfaceRegistry that implements the types.InterfaceRegistry interface
	mIRegistry := testfiles.NewMockInterfaceRegistry(ctrl)

	// Setup mock plugin behavior
	mockPlugin := new(mock.Mock)
	mockPlugin.On("Lookup", "cosmos_bank_v1beta1_RegisterLegacyAminoCodec").Return(func(c *codec.LegacyAmino) {}, nil)
	mockPlugin.On("Lookup", "cosmos_bank_v1beta1_RegisterInterfaces").Return(func(reg types.InterfaceRegistry) {}, nil)

	testCases := []testCase{
		{
			name:         "Empty unregistered types",
			ctxMock:      &clientCtx,
			pluginsDir:   "./testfiles",
			unregistered: map[string]struct{}{},
			mockFiles:    []string{"/path/to/plugins/plugin1.so"},
			mockLegacy: func(m *testfiles.MockCodec) error {
				return nil
			},
			mockRegistry: func(m *testfiles.MockInterfaceRegistry) error {
				m.EXPECT().RegisterImplementations(gomock.Any(), gomock.Any()).AnyTimes()
				return nil
			},
			expectedErr: nil,
		},
		{
			name:         "Unregistered type found",
			ctxMock:      &clientCtx,
			pluginsDir:   "./testfiles",
			unregistered: map[string]struct{}{"/cosmos.bank.v1beta1.MsgSend": {}},
			mockFiles:    []string{"/path/to/plugins/plugin1.so"},
			mockLegacy: func(m *testfiles.MockCodec) error {
				return nil
			},
			mockRegistry: func(m *testfiles.MockInterfaceRegistry) error {
				m.EXPECT().RegisterImplementations(gomock.Any(), gomock.Any()).AnyTimes()
				return nil
			},
			expectedErr: nil,
		},
		{
			name:         "Unregistered type not found",
			ctxMock:      &clientCtx,
			pluginsDir:   "./testfiles",
			unregistered: map[string]struct{}{"/cosmos.unknown.v1beta1.MsgUnknown": {}},
			mockFiles:    []string{"/path/to/plugins/plugin1.so"},
			mockLegacy: func(m *testfiles.MockCodec) error {
				return nil
			},
			mockRegistry: func(m *testfiles.MockInterfaceRegistry) error {
				m.EXPECT().RegisterImplementations(gomock.Any(), gomock.Any()).AnyTimes()
				return nil
			},
			expectedErr: fmt.Errorf("failed to lookup symbol /cosmos.unknown.v1beta1"),
		},
	}

	for idx, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set expectations for InterfaceRegistry method
			mCodec.EXPECT().InterfaceRegistry().Return(mIRegistry).AnyTimes()

			if tc.mockLegacy != nil {
				tc.mockLegacy(mCodec)
			}
			if tc.mockRegistry != nil {
				tc.mockRegistry(mIRegistry)
			}

			// Run the test
			err := RegisterTypes(*tc.ctxMock, tc.pluginsDir, tc.unregistered)
			if tc.expectedErr == nil && err != nil {
				t.Errorf("%d, Expected no error, got: %v", idx, err)
			} else if tc.expectedErr != nil && err == nil {
				t.Errorf("%d, Expected error: %v, got nil", idx, tc.expectedErr)
			} else if tc.expectedErr != nil && err != nil {
				if err.Error() != tc.expectedErr.Error() {
					t.Errorf("%d, Expected error: %v, got: %v", idx, tc.expectedErr, err)
				}
			} else if err == nil && tc.expectedErr == nil {
				// No error expected
			} else {
				t.Error("Should not get here")
			}
		})
	}
}
