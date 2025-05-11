package blockrelay

import (
	"github.com/Hoosat-Oy/HTND/app/appmessage"
	"github.com/Hoosat-Oy/HTND/app/protocol/protocolerrors"
	"github.com/Hoosat-Oy/HTND/domain"
	"github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter/router"
	"github.com/pkg/errors"
)

// HandleIBDBlockRequestsContext is the interface for the context needed for the HandleIBDBlockRequests flow.
type HandleIBDBlockRequestsContext interface {
	Domain() domain.Domain
}

// HandleIBDBlockRequests listens to appmessage.MsgRequestRelayBlocks messages and sends
// their corresponding blocks to the requesting peer.
func HandleIBDBlockRequests(context HandleIBDBlockRequestsContext, incomingRoute *router.Route,
	outgoingRoute *router.Route) error {

	for {
		message, err := incomingRoute.Dequeue()
		if err != nil {
			return err
		}
		msgRequestIBDBlocks := message.(*appmessage.MsgRequestIBDBlocks)
		log.Debugf("Got request for %d ibd blocks", len(msgRequestIBDBlocks.Hashes))

		for i := 0; i < len(msgRequestIBDBlocks.Hashes); i++ {
			// Fetch the block from the database.
			block, found, err := context.Domain().Consensus().GetBlock(msgRequestIBDBlocks.Hashes[i])
			if err != nil {
				return errors.Wrapf(err, "unable to fetch requested block hash %s", msgRequestIBDBlocks.Hashes[i])
			}

			if !found {
				return protocolerrors.Errorf(false, "IBD block %s not found", msgRequestIBDBlocks.Hashes[i])
			}

			// TODO (Partial nodes): Convert block to partial block if needed

			blockMessage := appmessage.DomainBlockToMsgBlock(block)
			ibdBlockMessage := appmessage.NewMsgIBDBlock(blockMessage)
			err = outgoingRoute.Enqueue(ibdBlockMessage)
			if err != nil {
				return err
			}
			log.Debugf("sent %d out of %d", i+1, len(msgRequestIBDBlocks.Hashes))
		}
	}
}
