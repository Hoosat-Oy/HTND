package rpchandlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Hoosat-Oy/HTND/app/appmessage"
	"github.com/Hoosat-Oy/HTND/app/protocol/protocolerrors"
	"github.com/Hoosat-Oy/HTND/app/rpc/rpccontext"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/ruleerrors"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/consensushashing"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/constants"
	"github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter/router"
	"github.com/pkg/errors"
)

// HandleSubmitBlock handles the respectively named RPC command
func HandleSubmitBlock(context *rpccontext.Context, _ *router.Router, request appmessage.Message) (appmessage.Message, error) {
	submitBlockRequest := request.(*appmessage.SubmitBlockRequestMessage)
	var err error
	var powHash *externalapi.DomainHash
	daaScore := submitBlockRequest.Block.Header.DAAScore
	var version uint16 = 1
	for _, powScore := range context.Config.ActiveNetParams.POWScores {
		if daaScore >= powScore {
			version = version + 1
		}
	}
	constants.BlockVersion = version
	if submitBlockRequest.Block.Header.Version >= 3 && constants.BlockVersion >= 3 {
		if submitBlockRequest.PowHash == "" {
			submitBlockRequestJSON, _ := json.MarshalIndent(submitBlockRequest.Block, "", "    ")
			return &appmessage.SubmitBlockResponseMessage{
				Error:        appmessage.RPCErrorf(fmt.Sprintf("Block not submitted, proof of work missing! %s", string(submitBlockRequestJSON))),
				RejectReason: appmessage.RejectReasonBlockInvalid,
			}, nil
		}
		powHash, err = externalapi.NewDomainHashFromString(strings.Replace(submitBlockRequest.PowHash, "0x", "", 1))
		if err != nil {
			submitBlockRequestJSON, _ := json.MarshalIndent(submitBlockRequest.Block, "", "    ")
			return &appmessage.SubmitBlockResponseMessage{
				Error:        appmessage.RPCErrorf(fmt.Sprintf("lock not submitted, proof of work is not valid data! %s", string(submitBlockRequestJSON))),
				RejectReason: appmessage.RejectReasonBlockInvalid,
			}, nil
		}
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

	domainBlock, err := appmessage.RPCBlockToDomainBlock(submitBlockRequest.Block)
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
	err = context.ProtocolManager.AddBlock(domainBlock, powHash)
	if err != nil {
		isProtocolOrRuleError := errors.As(err, &ruleerrors.RuleError{}) || errors.As(err, &protocolerrors.ProtocolError{})
		if !isProtocolOrRuleError {
			return nil, err
		}

		submitBlockRequestJSON, _ := json.MarshalIndent(submitBlockRequest.Block, "", "    ")
		if submitBlockRequestJSON != nil {
			log.Warnf("The RPC submitted block triggered a rule/protocol error (%s), printing "+
				"the full block for debug purposes: \n%s", err, string(submitBlockRequestJSON))
		}

		return &appmessage.SubmitBlockResponseMessage{
			Error:        appmessage.RPCErrorf("Block rejected. Reason: %s", err),
			RejectReason: appmessage.RejectReasonBlockInvalid,
		}, nil
	}

	log.Infof("Accepted block %s via submitBlock", consensushashing.BlockHash(domainBlock))

	response := appmessage.NewSubmitBlockResponseMessage()
	return response, nil
}
