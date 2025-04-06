package rpchandlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Hoosat-Oy/HTND/app/appmessage"
	"github.com/Hoosat-Oy/HTND/app/protocol/protocolerrors"
	"github.com/Hoosat-Oy/HTND/app/rpc/rpccontext"
	"github.com/Hoosat-Oy/HTND/domain/consensus/ruleerrors"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/consensushashing"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/constants"
	"github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter/router"
	"github.com/pkg/errors"
)

// HandleSubmitBlock handles the respectively named RPC command
func HandleSubmitBlock(context *rpccontext.Context, router *router.Router, request appmessage.Message) (appmessage.Message, error) {
	submitBlockRequest := request.(*appmessage.SubmitBlockRequestMessage)
	var err error
	var powHash string
	daaScore := submitBlockRequest.Block.Header.DAAScore
	var version uint16 = 1
	for _, powScore := range context.Config.ActiveNetParams.POWScores {
		if daaScore >= powScore {
			version = version + 1
		}
	}
	constants.BlockVersion = version
	if submitBlockRequest.Block.Header.Version != uint32(constants.BlockVersion) {
		submitBlockRequestJSON, _ := json.MarshalIndent(submitBlockRequest.Block, "", "    ")
		return &appmessage.SubmitBlockResponseMessage{
			Error:        appmessage.RPCErrorf(fmt.Sprintf("Block not submitted, wrong block version! %s", string(submitBlockRequestJSON))),
			RejectReason: appmessage.RejectReasonBlockInvalid,
		}, nil
	}
	if constants.BlockVersion >= constants.PoWIntegrityMinVersion {
		if submitBlockRequest.PowHash == "" {
			submitBlockRequestJSON, _ := json.MarshalIndent(submitBlockRequest.Block, "", "    ")
			return &appmessage.SubmitBlockResponseMessage{
				Error:        appmessage.RPCErrorf(fmt.Sprintf("Block not submitted, proof of work missing! %s", string(submitBlockRequestJSON))),
				RejectReason: appmessage.RejectReasonBlockInvalid,
			}, nil
		}
		powHash = strings.Replace(submitBlockRequest.PowHash, "0x", "", 1)
	}

	isSynced := false
	// The node is considered synced if it has peers and consensus state is nearly synced
	if context.ProtocolManager.Context().HasPeers() {
		isSynced, err = context.ProtocolManager.Context().IsNearlySynced()
		if err != nil {
			return nil, err
		}
	}

	if !context.Config.AllowSubmitBlockWhenNotSynced && !isSynced {
		return &appmessage.SubmitBlockResponseMessage{
			Error:        appmessage.RPCErrorf("Block not submitted - node is not synced"),
			RejectReason: appmessage.RejectReasonIsInIBD,
		}, nil
	}

	domainBlock, err := appmessage.RPCBlockToDomainBlock(submitBlockRequest.Block, powHash)
	if err != nil {
		return &appmessage.SubmitBlockResponseMessage{
			Error:        appmessage.RPCErrorf("Could not parse block: %s", err),
			RejectReason: appmessage.RejectReasonBlockInvalid,
		}, nil
	}

	if !submitBlockRequest.AllowNonDAABlocks {
		virtualDAAScore, err := context.Domain.Consensus().GetVirtualDAAScore()
		if err != nil {
			return nil, err
		}
		// A simple heuristic check which signals that the mined block is out of date
		// and should not be accepted unless user explicitly requests
		daaWindowSize := uint64(context.Config.NetParams().DifficultyAdjustmentWindowSize)
		if virtualDAAScore > daaWindowSize && domainBlock.Header.DAAScore() < virtualDAAScore-daaWindowSize {
			return &appmessage.SubmitBlockResponseMessage{
				Error: appmessage.RPCErrorf("Block rejected. Reason: block DAA score %d is too far "+
					"behind virtual's DAA score %d", domainBlock.Header.DAAScore(), virtualDAAScore),
				RejectReason: appmessage.RejectReasonBlockInvalid,
			}, nil
		}
	}
	err = context.ProtocolManager.AddBlock(domainBlock)
	if err != nil {
		isProtocolOrRuleError := errors.As(err, &ruleerrors.RuleError{}) || errors.As(err, &protocolerrors.ProtocolError{})
		if !isProtocolOrRuleError {
			return nil, err
		}

		if errors.Is(err, ruleerrors.ErrInvalidPoW) {
			submitBlockRequestJSON, _ := json.MarshalIndent(submitBlockRequest.Block, "", "    ")
			if submitBlockRequestJSON != nil {
				log.Warnf("The RPC submitted block triggered a invalid PoW error (%s), consider banning.", powHash)
				// Consider implementing banning for RPC, though works network wide on P2P.
				// If node receives submission with wrong PoW hash and submits them forward the node will get banned.
				// So pool that accepts invalid PoW has submissions will get banned from the network.
			}
		} else {
			submitBlockRequestJSON, _ := json.MarshalIndent(submitBlockRequest.Block, "", "    ")
			if submitBlockRequestJSON != nil {
				log.Warnf("The RPC submitted block triggered a rule/protocol error (%s), printing "+
					"the full block for debug purposes: \n%s", err, string(submitBlockRequestJSON))
			}
		}

		return &appmessage.SubmitBlockResponseMessage{
			Error:        appmessage.RPCErrorf("Block rejected. Reason: %s", err),
			RejectReason: appmessage.RejectReasonBlockInvalid,
		}, nil
	}

	log.Infof("Accepted block %s via submitBlock", consensushashing.BlockHash(domainBlock))
	log.Infof("Accepted PoW hash %s", domainBlock.PoWHash)

	response := appmessage.NewSubmitBlockResponseMessage()
	return response, nil
}
