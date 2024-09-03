package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	rpcclientmock "github.com/cometbft/cometbft/rpc/client/mock"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	kmultisig "github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	testutilmod "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/stretchr/testify/suite"
)

func TxSignExec(clientCtx client.Context, from sdk.AccAddress, keyringDir string, filename string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := []string{
		fmt.Sprintf("--from=%s", from.String()),
		fmt.Sprintf("--%s=%s", flags.FlagHome, strings.Replace(clientCtx.HomeDir, "simd", "simcli", 1)),
		fmt.Sprintf("--%s=%s", flags.FlagChainID, clientCtx.ChainID),
		fmt.Sprintf("--keyring-backend=%s", clientCtx.Keyring.Backend()),
		fmt.Sprintf("--keyring-dir=%s", keyringDir),
		"--plugins-dir=./testfiles/",
		"-a=0",
		"--sequence=2",
		"--offline",
		filename,
	}

	cmd := GetSignCommand()
	cmd.PersistentFlags().String(flags.FlagHome, clientCtx.HomeDir, "directory for config and data")

	return clitestutil.ExecTestCLICmd(clientCtx, cmd, append(args, extraArgs...))
}

func TxValidateSignaturesExec(clientCtx client.Context, keyringDir string, filename string) (testutil.BufferWriter, error) {
	args := []string{
		filename,
		fmt.Sprintf("--%s=%s", flags.FlagChainID, clientCtx.ChainID),
		"--offline",
		fmt.Sprintf("--keyring-backend=%s", clientCtx.Keyring.Backend()),
		fmt.Sprintf("--keyring-dir=%s", keyringDir),
		"--plugins-dir=./testfiles/",
		"-a=0",
		"--sequence=2",
	}

	return clitestutil.ExecTestCLICmd(clientCtx, GetValidateSignaturesCommand(), args)
}

func TxSignBatchExec(clientCtx client.Context, from fmt.Stringer, keyringDir string, filename string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := []string{
		fmt.Sprintf("--from=%s", from.String()),
		fmt.Sprintf("--%s=%s", flags.FlagChainID, clientCtx.ChainID),
		fmt.Sprintf("--keyring-backend=%s", clientCtx.Keyring.Backend()),
		fmt.Sprintf("--keyring-dir=%s", keyringDir),
		"--plugins-dir=./testfiles/",
		"-a=0",
		"--sequence=2",
		"--offline",
		filename,
	}

	return clitestutil.ExecTestCLICmd(clientCtx, GetSignBatchCommand(), append(args, extraArgs...))
}

func TxMultiSignExec(clientCtx client.Context, from, keyringDir, filename string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := []string{
		fmt.Sprintf("--%s=%s", flags.FlagChainID, clientCtx.ChainID),
		fmt.Sprintf("--keyring-backend=%s", clientCtx.Keyring.Backend()),
		fmt.Sprintf("--keyring-dir=%s", keyringDir),
		"--plugins-dir=./testfiles/",
		"-a=0",
		"--sequence=2",
		"--offline",
		filename,
		from,
	}

	return clitestutil.ExecTestCLICmd(clientCtx, GetMultiSignCommand(), append(args, extraArgs...))
}

type CLITestSuite struct {
	suite.Suite

	kr        keyring.Keyring
	encCfg    testutilmod.TestEncodingConfig
	baseCtx   client.Context
	clientCtx client.Context
	val       sdk.AccAddress
	val1      sdk.AccAddress

	keyringDir string

	ac address.Codec
}

func (s *CLITestSuite) createBankMsg(clientCtx client.Context, toAddr sdk.AccAddress, amount sdk.Coins, extraFlags ...string) (testutil.BufferWriter, error) {
	flags := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees,
			sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(10))).String()),
	}

	flags = append(flags, extraFlags...)
	return clitestutil.MsgSendExec(clientCtx, s.val, toAddr, amount, s.ac, flags...)
}

func TestCLITestSuite(t *testing.T) {
	suite.Run(t, new(CLITestSuite))
}

func (s *CLITestSuite) TearDownSuite() {
	os.RemoveAll(s.keyringDir)
}

func (s *CLITestSuite) SetupSuite() {
	s.encCfg = testutilmod.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, gov.AppModuleBasic{})
	keyringDir, err := os.MkdirTemp("", fmt.Sprintf("tmpKeyringDir-%s", time.Now().Format("20060102-150405")))
	s.Require().NoError(err)
	s.keyringDir = keyringDir

	kr, err := keyring.New("test", keyring.BackendTest, keyringDir, nil, s.encCfg.Codec)
	s.Require().NoError(err)
	s.kr = kr
	s.baseCtx = client.Context{}.
		WithKeyring(s.kr).
		WithTxConfig(s.encCfg.TxConfig).
		WithCodec(s.encCfg.Codec).
		WithClient(clitestutil.MockCometRPC{Client: rpcclientmock.Client{}}).
		WithAccountRetriever(client.MockAccountRetriever{}).
		WithOutput(io.Discard).
		WithChainID("test-chain")

	ctxGen := func() client.Context {
		bz, _ := s.encCfg.Codec.Marshal(&sdk.TxResponse{})
		c := clitestutil.NewMockCometRPC(abci.ResponseQuery{
			Value: bz,
		})
		return s.baseCtx.WithClient(c)
	}
	s.clientCtx = ctxGen()

	kb := s.clientCtx.Keyring
	valAcc, _, err := kb.NewMnemonic("newAccount", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
	s.Require().NoError(err)
	s.val, err = valAcc.GetAddress()
	s.Require().NoError(err)

	account1, _, err := kb.NewMnemonic("newAccount1", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
	s.Require().NoError(err)
	s.val1, err = account1.GetAddress()
	s.Require().NoError(err)

	account2, _, err := kb.NewMnemonic("newAccount2", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
	s.Require().NoError(err)
	pub1, err := account1.GetPubKey()
	s.Require().NoError(err)
	pub2, err := account2.GetPubKey()
	s.Require().NoError(err)

	// Create a dummy account for testing purpose
	_, _, err = kb.NewMnemonic("dummyAccount", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
	s.Require().NoError(err)

	multi := kmultisig.NewLegacyAminoPubKey(2, []cryptotypes.PubKey{pub1, pub2})
	_, err = kb.SaveMultisig("multi", multi)
	s.Require().NoError(err)

	s.ac = addresscodec.NewBech32Codec("cosmos")
}

func (s *CLITestSuite) TestCLIValidateSignatures() {
	sendTokens := sdk.NewCoins(
		sdk.NewCoin("testtoken", math.NewInt(10)),
		sdk.NewCoin("stake", math.NewInt(10)))

	res, err := s.createBankMsg(s.clientCtx, s.val, sendTokens,
		fmt.Sprintf("--%s=true", flags.FlagGenerateOnly))
	s.Require().NoError(err)

	// write  unsigned tx to file
	unsignedTx := testutil.WriteToNewTempFile(s.T(), res.String())
	defer unsignedTx.Close()

	res, err = TxSignExec(s.clientCtx, s.val, s.keyringDir, unsignedTx.Name())
	s.Require().NoError(err)
	signedTx, err := s.clientCtx.TxConfig.TxJSONDecoder()(res.Bytes())
	s.Require().NoError(err)

	// write signed tx to file
	signedTxFile := testutil.WriteToNewTempFile(s.T(), res.String())
	defer signedTxFile.Close()

	// verify signed tx
	// txBuilder, err := s.clientCtx.TxConfig.WrapTxBuilder(signedTx)
	_, err = s.clientCtx.TxConfig.WrapTxBuilder(signedTx)
	s.Require().NoError(err)
	_, err = TxValidateSignaturesExec(s.clientCtx, s.keyringDir, signedTxFile.Name())
	s.Require().NoError(err)

	// TODO: modifying the tx and verifying isn't working. It always passes when it should fail...
	// modify tx and fail
	// txBuilder.SetMemo("MODIFIED TX")
	// bz, err := s.clientCtx.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
	// s.Require().NoError(err)

	// modifiedTxFile := testutil.WriteToNewTempFile(s.T(), string(bz))
	// defer modifiedTxFile.Close()
	// _, err = TxValidateSignaturesExec(s.clientCtx, s.keyringDir, modifiedTxFile.Name())
	// note: this is currently failing. No error is being thrown when one is expected.
	// TODO(adam-hanna): consult with @giunatale or @tbruyelle to fix
	//s.Require().EqualError(err, "signatures validation failed")
}

func (s *CLITestSuite) TestCLISignBatch() {
	sendTokens := sdk.NewCoins(
		sdk.NewCoin("testtoken", math.NewInt(10)),
		sdk.NewCoin("stake", math.NewInt(10)),
	)

	generatedStd, err := s.createBankMsg(s.clientCtx, s.val,
		sendTokens, fmt.Sprintf("--%s=true", flags.FlagGenerateOnly))
	s.Require().NoError(err)

	outputFile := testutil.WriteToNewTempFile(s.T(), strings.Repeat(generatedStd.String(), 3))
	defer outputFile.Close()
	s.clientCtx.HomeDir = strings.Replace(s.clientCtx.HomeDir, "simd", "simcli", 1)

	// sign-batch file - offline is set but account-number and sequence are not
	_, err = TxSignBatchExec(s.clientCtx, s.val, s.keyringDir, outputFile.Name(), fmt.Sprintf("--%s=%s", flags.FlagChainID, s.clientCtx.ChainID))
	s.Require().NoError(err)
}

func (s *CLITestSuite) TestCLIMultisign() {
	// Generate 2 accounts and a multisig.
	account1, err := s.clientCtx.Keyring.Key("newAccount1")
	s.Require().NoError(err)

	account2, err := s.clientCtx.Keyring.Key("newAccount2")
	s.Require().NoError(err)

	multisigRecord, err := s.clientCtx.Keyring.Key("multi")
	s.Require().NoError(err)

	addr, err := multisigRecord.GetAddress()
	s.Require().NoError(err)

	// Generate multisig transaction.
	multiGeneratedTx, err := clitestutil.MsgSendExec(
		s.clientCtx,
		addr,
		s.val,
		sdk.NewCoins(
			sdk.NewInt64Coin("stake", 5),
		),
		s.ac,
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(10))).String()),
		fmt.Sprintf("--%s=true", flags.FlagGenerateOnly),
	)
	s.Require().NoError(err)

	// Save tx to file
	multiGeneratedTxFile := testutil.WriteToNewTempFile(s.T(), multiGeneratedTx.String())
	defer multiGeneratedTxFile.Close()

	addr1, err := account1.GetAddress()
	s.Require().NoError(err)
	// Sign with account1
	s.clientCtx.HomeDir = strings.Replace(s.clientCtx.HomeDir, "simd", "simcli", 1)
	account1Signature, err := TxSignExec(s.clientCtx, addr1, s.keyringDir, multiGeneratedTxFile.Name(), "--multisig", addr.String())
	s.Require().NoError(err)

	sign1File := testutil.WriteToNewTempFile(s.T(), account1Signature.String())
	defer sign1File.Close()

	addr2, err := account2.GetAddress()
	s.Require().NoError(err)
	// Sign with account2
	account2Signature, err := TxSignExec(s.clientCtx, addr2, s.keyringDir, multiGeneratedTxFile.Name(), "--multisig", addr.String())
	s.Require().NoError(err)

	sign2File := testutil.WriteToNewTempFile(s.T(), account2Signature.String())
	defer sign2File.Close()

	s.clientCtx.Offline = false
	multiSigWith2Signatures, err := TxMultiSignExec(s.clientCtx, multisigRecord.Name, s.keyringDir, multiGeneratedTxFile.Name(), sign1File.Name(), sign2File.Name())
	s.Require().NoError(err)

	// Write the output to disk
	signedTxFile := testutil.WriteToNewTempFile(s.T(), multiSigWith2Signatures.String())
	defer signedTxFile.Close()

	_, err = TxValidateSignaturesExec(s.clientCtx, s.keyringDir, signedTxFile.Name())
	s.Require().NoError(err)
}

func (s *CLITestSuite) TestSignBatchMultisig() {
	// Fetch 2 accounts and a multisig.
	account1, err := s.clientCtx.Keyring.Key("newAccount1")
	s.Require().NoError(err)
	account2, err := s.clientCtx.Keyring.Key("newAccount2")
	s.Require().NoError(err)
	multisigRecord, err := s.clientCtx.Keyring.Key("multi")
	s.Require().NoError(err)

	addr, err := multisigRecord.GetAddress()
	s.Require().NoError(err)
	// Send coins from validator to multisig.
	sendTokens := sdk.NewInt64Coin("stake", 10)
	_, err = s.createBankMsg(
		s.clientCtx,
		addr,
		sdk.NewCoins(sendTokens),
	)
	s.Require().NoError(err)

	generatedStd, err := clitestutil.MsgSendExec(
		s.clientCtx,
		addr,
		s.val,
		sdk.NewCoins(
			sdk.NewCoin("stake", math.NewInt(1)),
		),
		s.ac,
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(10))).String()),
		fmt.Sprintf("--%s=true", flags.FlagGenerateOnly),
	)
	s.Require().NoError(err)

	// Write the output to disk
	filename := testutil.WriteToNewTempFile(s.T(), strings.Repeat(generatedStd.String(), 1))
	defer filename.Close()
	s.clientCtx.HomeDir = strings.Replace(s.clientCtx.HomeDir, "simd", "simcli", 1)

	addr1, err := account1.GetAddress()
	s.Require().NoError(err)
	// sign-batch file
	res, err := TxSignBatchExec(s.clientCtx, addr1, s.keyringDir, filename.Name(), "--multisig", addr.String(), "--signature-only")
	s.Require().NoError(err)
	s.Require().Equal(1, len(strings.Split(strings.Trim(res.String(), "\n"), "\n")))
	// write sigs to file
	file1 := testutil.WriteToNewTempFile(s.T(), res.String())
	defer file1.Close()

	addr2, err := account2.GetAddress()
	s.Require().NoError(err)
	// sign-batch file with account2
	res, err = TxSignBatchExec(s.clientCtx, addr2, s.keyringDir, filename.Name(), "--multisig", addr.String(), "--signature-only")
	s.Require().NoError(err)
	s.Require().Equal(1, len(strings.Split(strings.Trim(res.String(), "\n"), "\n")))
	// write sigs to file2
	file2 := testutil.WriteToNewTempFile(s.T(), res.String())
	defer file2.Close()
	_, err = TxMultiSignExec(s.clientCtx, multisigRecord.Name, s.keyringDir, filename.Name(), file1.Name(), file2.Name())
	s.Require().NoError(err)
}
