package blockrelay

import (
	"sync"

	"github.com/Hoosat-Oy/HTND/app/appmessage"
	"github.com/Hoosat-Oy/HTND/app/protocol/protocolerrors"
	"github.com/Hoosat-Oy/HTND/domain"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
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

		var wg sync.WaitGroup
		errChan := make(chan error, len(msgRequestIBDBlocks.Hashes))
		for i := 0; i < len(msgRequestIBDBlocks.Hashes); i++ {
			hash := msgRequestIBDBlocks.Hashes[i]
			wg.Add(1)
			go func(hash *externalapi.DomainHash) {
				defer wg.Done()
				// Fetch the block from the database.
				block, found, err := context.Domain().Consensus().GetBlock(hash)
				if err != nil {
					errChan <- errors.Wrapf(err, "unable to fetch requested block hash %s", hash)
					return
				}

				if !found {
					errChan <- protocolerrors.Errorf(false, "IBD block %s not found", hash)
					return
				}

				// TODO (Partial nodes): Convert block to partial block if needed

				blockMessage := appmessage.DomainBlockToMsgBlock(block)
				ibdBlockMessage := appmessage.NewMsgIBDBlock(blockMessage)
				err = outgoingRoute.Enqueue(ibdBlockMessage)
				if err != nil {
					errChan <- err
					return
				}
				log.Debugf("sent %d out of %d", i+1, len(msgRequestIBDBlocks.Hashes))
			}(hash)
		}

		wg.Wait()
		close(errChan)

		for err := range errChan {
			if err != nil {
				return err
			}
		}
	}
}
