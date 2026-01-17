package testutils

import (
	"sync"
	"testing"
	"time"

	"github.com/Hoosat-Oy/HTND/domain/consensus"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/constants"
	"github.com/Hoosat-Oy/HTND/domain/dagconfig"
)

var blockVersionTestLock sync.Mutex

func cloneParams(params dagconfig.Params) dagconfig.Params {
	cloned := params
	cloned.DNSSeeds = append([]string(nil), params.DNSSeeds...)
	cloned.K = append([]externalapi.KType(nil), params.K...)
	cloned.TargetTimePerBlock = append([]time.Duration(nil), params.TargetTimePerBlock...)
	cloned.FinalityDuration = append([]time.Duration(nil), params.FinalityDuration...)
	cloned.DifficultyAdjustmentWindowSize = append([]int(nil), params.DifficultyAdjustmentWindowSize...)
	cloned.PruningMultiplier = append([]uint64(nil), params.PruningMultiplier...)
	cloned.MaxBlockMass = append([]uint64(nil), params.MaxBlockMass...)
	cloned.MaxBlockParents = append([]externalapi.KType(nil), params.MaxBlockParents...)
	cloned.MergeDepth = append([]uint64(nil), params.MergeDepth...)
	cloned.POWScores = append([]uint64(nil), params.POWScores...)
	return cloned
}

// ForAllNets runs the passed testFunc with all available networks
// if setDifficultyToMinumum = true - will modify the net params to have minimal difficulty, like in SimNet
func ForAllNets(t *testing.T, skipPow bool, testFunc func(*testing.T, *consensus.Config)) {
	allParams := []dagconfig.Params{
		dagconfig.MainnetParams,
		dagconfig.TestnetParams,
	}

	for _, params := range allParams {
		consensusConfig := consensus.Config{Params: cloneParams(params)}
		t.Run(consensusConfig.Name, func(t *testing.T) {
			blockVersionTestLock.Lock()
			defer blockVersionTestLock.Unlock()
			// NOTE: Do not run these subtests in parallel.
			// The consensus code mutates the global block version via constants.SetBlockVersion
			// during validation/building, so parallel subtests will interfere with each other.
			previousBlockVersion := constants.GetBlockVersion()
			constants.SetBlockVersion(1)
			t.Cleanup(func() {
				constants.SetBlockVersion(previousBlockVersion)
			})
			consensusConfig.SkipProofOfWork = skipPow
			t.Logf("Running test for %s", consensusConfig.Name)
			testFunc(t, &consensusConfig)
		})
	}
}
