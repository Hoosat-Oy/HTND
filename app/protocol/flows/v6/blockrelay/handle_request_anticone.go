package blockrelay

import (
	"sort"

	"github.com/Hoosat-Oy/HTND/app/appmessage"
	"github.com/Hoosat-Oy/HTND/app/protocol/peer"
	"github.com/Hoosat-Oy/HTND/app/protocol/protocolerrors"
	"github.com/Hoosat-Oy/HTND/domain"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/infrastructure/config"
	"github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter/router"
)

// RequestAnticoneContext is the interface for the context needed for the HandleRequestHeaders flow.
type RequestAnticoneContext interface {
	Domain() domain.Domain
	Config() *config.Config
}

type handleRequestAnticoneFlow struct {
	RequestAnticoneContext
	incomingRoute, outgoingRoute *router.Route
	peer                         *peer.Peer
}

// HandleRequestAnticone handles RequestAnticone messages
func HandleRequestAnticone(context RequestAnticoneContext, incomingRoute *router.Route,
	outgoingRoute *router.Route, peer *peer.Peer) error {

	flow := &handleRequestAnticoneFlow{
		RequestAnticoneContext: context,
		incomingRoute:          incomingRoute,
		outgoingRoute:          outgoingRoute,
		peer:                   peer,
	}
	return flow.start()
}

func (flow *handleRequestAnticoneFlow) start() error {
	for {
		blockHash, contextHash, err := receiveRequestAnticone(flow.incomingRoute)
		if err != nil {
			return err
		}
		// TODO: disable to check if it's still needed
		// if blockHash.Equal(contextHash) {
		// 	// response with empy headers if the blockhash is the same as the contexthash.
		// 	log.Debugf("Received requestAnticone with blockHash: %s, contextHash: %s, sending empty headers", blockHash, contextHash)
		// 	emptyHeaders := make([]*appmessage.MsgBlockHeader, 0)
		// 	blockHeadersMessage := appmessage.NewBlockHeadersMessage(emptyHeaders)
		// 	err = flow.outgoingRoute.Enqueue(blockHeadersMessage)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	err = flow.outgoingRoute.Enqueue(appmessage.NewMsgDoneHeaders())
		// 	if err != nil {
		// 		return err
		// 	}
		// 	continue
		// }
		log.Debugf("Received requestAnticone with blockHash: %s, contextHash: %s", blockHash, contextHash)
		log.Debugf("Getting past(%s) cap anticone(%s) for peer %s", contextHash, blockHash, flow.peer)

		// GetAnticone is expected to be called by the syncee for getting the anticone of the header selected tip
		// intersected by past of relayed block, and is thus expected to be bounded by mergeset limit since
		// we relay blocks only if they enter virtual's mergeset. We add a 2 factor for possible sync gaps.
		var blockHashes []*externalapi.DomainHash
		blockHashes, err = flow.Domain().Consensus().GetAnticone(blockHash, contextHash, flow.Config().ActiveNetParams.MergeSetSizeLimit*1000)
		if err != nil {
			return protocolerrors.Wrap(true, err, "Failed querying anticone")
		}
		log.Debugf("Got %d header hashes in past(%s) cap anticone(%s)", len(blockHashes), contextHash, blockHash)

		blockHeaders := make([]*appmessage.MsgBlockHeader, len(blockHashes))
		for i, blockHash := range blockHashes {
			blockHeader, err := flow.Domain().Consensus().GetBlockHeader(blockHash)
			if err != nil {
				return err
			}
			blockHeaders[i] = appmessage.DomainBlockHeaderToBlockHeader(blockHeader)
		}

		// We sort the headers in bottom-up topological order before sending
		sort.Slice(blockHeaders, func(i, j int) bool {
			return blockHeaders[i].BlueWork.Cmp(blockHeaders[j].BlueWork) < 0
		})

		blockHeadersMessage := appmessage.NewBlockHeadersMessage(blockHeaders)
		err = flow.outgoingRoute.Enqueue(blockHeadersMessage)
		if err != nil {
			return err
		}

		err = flow.outgoingRoute.Enqueue(appmessage.NewMsgDoneHeaders())
		if err != nil {
			return err
		}
	}
}

func receiveRequestAnticone(incomingRoute *router.Route) (blockHash *externalapi.DomainHash,
	contextHash *externalapi.DomainHash, err error) {

	message, err := incomingRoute.Dequeue()
	if err != nil {
		return nil, nil, err
	}
	msgRequestAnticone := message.(*appmessage.MsgRequestAnticone)

	return msgRequestAnticone.BlockHash, msgRequestAnticone.ContextHash, nil
}
