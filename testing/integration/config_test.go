package integration

import (
	"os"
	"testing"
	"time"

	"github.com/Hoosat-Oy/HTND/domain/dagconfig"
	"github.com/Hoosat-Oy/HTND/infrastructure/config"
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

	miningAddress1           = "hoosatsim:qznzhj4n5dd796qrqzueryfglvnesxpl5fywr9c5057ndvgx7y0j7x92q58rs"
	miningAddress1PrivateKey = "hoosatsim:qznzhj4n5dd796qrqzueryfglvnesxpl5fywr9c5057ndvgx7y0j7x92q58rs"

	miningAddress2           = "hoosatsim:qznzhj4n5dd796qrqzueryfglvnesxpl5fywr9c5057ndvgx7y0j7x92q58rs"
	miningAddress2PrivateKey = "hoosatsim:qznzhj4n5dd796qrqzueryfglvnesxpl5fywr9c5057ndvgx7y0j7x92q58rs"

	miningAddress3           = "hoosatsim:qznzhj4n5dd796qrqzueryfglvnesxpl5fywr9c5057ndvgx7y0j7x92q58rs"
	miningAddress3PrivateKey = "955da5fe765a921d22ccba5102a31f3b893b79607e48195c3d63a795486473ba"

	defaultTimeout = 30 * time.Second
)

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
}

func commonConfig() *config.Config {
	commonConfig := config.DefaultConfig()

	*commonConfig.ActiveNetParams = dagconfig.SimnetParams // Copy so that we can make changes safely
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
