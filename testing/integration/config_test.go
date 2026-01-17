package integration

import (
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/kaspanet/go-secp256k1"

	"github.com/Hoosat-Oy/HTND/domain/dagconfig"
	"github.com/Hoosat-Oy/HTND/infrastructure/config"
	"github.com/Hoosat-Oy/HTND/util"
)

const (
	p2pAddress1 = "127.0.0.1:45321"
	p2pAddress2 = "127.0.0.1:45322"
	p2pAddress3 = "127.0.0.1:45323"
	p2pAddress4 = "127.0.0.1:45324"
	p2pAddress5 = "127.0.0.1:45325"

	rpcAddress1 = "127.0.0.1:21345"
	rpcAddress2 = "127.0.0.1:21346"
	rpcAddress3 = "127.0.0.1:21347"
	rpcAddress4 = "127.0.0.1:21348"
	rpcAddress5 = "127.0.0.1:21349"

	defaultTimeout = 30 * time.Second
)

// NOTE: Integration tests need mining address private keys that are real schnorr
// private keys (32-byte hex), because some tests sign spends from coinbase UTXOs.
// Keep these deterministic.
var (
	miningAddress1PrivateKey = "0000000000000000000000000000000000000000000000000000000000000001"
	miningAddress2PrivateKey = "0000000000000000000000000000000000000000000000000000000000000002"
	miningAddress3PrivateKey = "0000000000000000000000000000000000000000000000000000000000000003"

	miningAddress1 = mustSchnorrAddressFromPrivateKeyHex(miningAddress1PrivateKey)
	miningAddress2 = mustSchnorrAddressFromPrivateKeyHex(miningAddress2PrivateKey)
	miningAddress3 = mustSchnorrAddressFromPrivateKeyHex(miningAddress3PrivateKey)
)

func mustSchnorrAddressFromPrivateKeyHex(privateKeyHex string) string {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		panic(err)
	}
	keyPair, err := secp256k1.DeserializeSchnorrPrivateKeyFromSlice(privateKeyBytes)
	if err != nil {
		panic(err)
	}
	publicKey, err := keyPair.SchnorrPublicKey()
	if err != nil {
		panic(err)
	}
	publicKeySerialized, err := publicKey.Serialize()
	if err != nil {
		panic(err)
	}
	addr, err := util.NewAddressPublicKey(publicKeySerialized[:], util.Bech32PrefixHoosatSim)
	if err != nil {
		panic(err)
	}
	return addr.EncodeAddress()
}

func setConfig(t *testing.T, harness *appHarness, protocolVersion uint32) {
	harness.config = commonConfig()
	harness.config.AppDir = randomDirectory(t)
	harness.config.Listeners = []string{harness.p2pAddress}
	harness.config.RPCListeners = []string{harness.rpcAddress}
	harness.config.UTXOIndex = harness.utxoIndex
	harness.config.AllowSubmitBlockWhenNotSynced = true
	if protocolVersion != 0 {
		harness.config.ProtocolVersion = protocolVersion
	}

	if harness.overrideDAGParams != nil {
		harness.config.ActiveNetParams = harness.overrideDAGParams
	}

	// Integration tests shouldn't burn CPU on PoW solving.
	harness.config.ActiveNetParams.SkipProofOfWork = true
}

func commonConfig() *config.Config {
	commonConfig := config.DefaultConfig()

	*commonConfig.ActiveNetParams = dagconfig.SimnetParams // Copy so that we can make changes safely
	commonConfig.ActiveNetParams.SkipProofOfWork = true
	commonConfig.ActiveNetParams.BlockCoinbaseMaturity = 10
	commonConfig.TargetOutboundPeers = 0
	commonConfig.DisableDNSSeed = true
	commonConfig.Simnet = true

	return commonConfig
}

func randomDirectory(t *testing.T) string {
	dir, err := os.MkdirTemp("", "integration-test")
	if err != nil {
		t.Fatalf("Error creating temporary directory for test: %+v", err)
	}

	return dir
}
