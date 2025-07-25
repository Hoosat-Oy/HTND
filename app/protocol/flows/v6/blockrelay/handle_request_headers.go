package blockrelay

import (
	"github.com/Hoosat-Oy/HTND/app/protocol/peer"
	"github.com/Hoosat-Oy/HTND/app/protocol/protocolerrors"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"

	"github.com/Hoosat-Oy/HTND/app/appmessage"
	"github.com/Hoosat-Oy/HTND/domain"
	"github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter/router"
)

// This constant must be equal at both syncer and syncee. Therefore, never (!!) change this constant unless a new p2p
// version is introduced. See `TestIBDBatchSizeLessThanRouteCapacity` as well.
func getIBDBatchSize() int {
	return 99 * 5
}

// RequestHeadersContext is the interface for the context needed for the HandleRequestHeaders flow.
type RequestHeadersContext interface {
	Domain() domain.Domain
}

type handleRequestHeadersFlow struct {
	RequestHeadersContext
	incomingRoute, outgoingRoute *router.Route
	peer                         *peer.Peer
}

// HandleRequestHeaders handles RequestHeaders messages
func HandleRequestHeaders(context RequestHeadersContext, incomingRoute *router.Route,
	outgoingRoute *router.Route, peer *peer.Peer) error {

	flow := &handleRequestHeadersFlow{
		RequestHeadersContext: context,
		incomingRoute:         incomingRoute,
		outgoingRoute:         outgoingRoute,
		peer:                  peer,
	}
	return flow.start()
}

func (flow *handleRequestHeadersFlow) start() error {
	for {
		lowHash, highHash, err := receiveRequestHeaders(flow.incomingRoute)
		if err != nil {
			return err
		}
		log.Debugf("Received requestHeaders with lowHash: %s, highHash: %s", lowHash, highHash)

		consensus := flow.Domain().Consensus()

		lowHashInfo, err := consensus.GetBlockInfo(lowHash)
		if err != nil {
			return err
		}
		if !lowHashInfo.HasHeader() {
			return protocolerrors.Errorf(true, "Block %s does not exist", lowHash)
		}

		highHashInfo, err := consensus.GetBlockInfo(highHash)
		if err != nil {
			return err
		}
		if !highHashInfo.HasHeader() {
			return protocolerrors.Errorf(true, "Block %s does not exist", highHash)
		}

		isLowSelectedAncestorOfHigh, err := consensus.IsInSelectedParentChainOf(lowHash, highHash)
		if err != nil {
			return err
		}
		if !isLowSelectedAncestorOfHigh {
			return protocolerrors.Errorf(true, "Expected %s to be on the selected chain of %s",
				lowHash, highHash)
		}

		for !lowHash.Equal(highHash) {
			log.Debugf("Getting block headers between %s and %s to %s", lowHash, highHash, flow.peer)

			// GetHashesBetween is a relatively heavy operation so we limit it
			// in order to avoid locking the consensus for too long
			// maxBlocks MUST be >= MergeSetSizeLimit + 1
			const maxBlocks = 1 << 12
			blockHashes, _, err := consensus.GetHashesBetween(lowHash, highHash, maxBlocks)
			if err != nil {
				return err
			}
			log.Debugf("Got %d header hashes above lowHash %s", len(blockHashes), lowHash)

			blockHeaders := make([]*appmessage.MsgBlockHeader, len(blockHashes))
			for i, blockHash := range blockHashes {
				blockHeader, err := consensus.GetBlockHeader(blockHash)
				if err != nil {
					return err
				}
				blockHeaders[i] = appmessage.DomainBlockHeaderToBlockHeader(blockHeader)
			}

			log.Infof("Relaying %d headers through IBD to peer %s", len(blockHeaders), flow.peer.Address())
			blockHeadersMessage := appmessage.NewBlockHeadersMessage(blockHeaders)
			err = flow.outgoingRoute.Enqueue(blockHeadersMessage)
			if err != nil {
				return err
			}

			message, err := flow.incomingRoute.Dequeue()
			if err != nil {
				return err
			}
			if _, ok := message.(*appmessage.MsgRequestNextHeaders); !ok {
				return protocolerrors.Errorf(true, "received unexpected message type. "+
					"expected: %s, got: %s", appmessage.CmdRequestNextHeaders, message.Command())
			}

			// The next lowHash is the last element in blockHashes
			lowHash = blockHashes[len(blockHashes)-1]
		}
		err = flow.outgoingRoute.Enqueue(appmessage.NewMsgDoneHeaders())
		if err != nil {
			return err
		}
	}
}

func receiveRequestHeaders(incomingRoute *router.Route) (lowHash *externalapi.DomainHash,
	highHash *externalapi.DomainHash, err error) {

	message, err := incomingRoute.Dequeue()
	if err != nil {
		return nil, nil, err
	}
	msgRequestIBDBlocks := message.(*appmessage.MsgRequestHeaders)

	return msgRequestIBDBlocks.LowHash, msgRequestIBDBlocks.HighHash, nil
}
