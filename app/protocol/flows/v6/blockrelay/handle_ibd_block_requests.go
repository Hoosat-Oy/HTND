package blockrelay

import (
	"github.com/Hoosat-Oy/HTND/app/appmessage"
	peerpkg "github.com/Hoosat-Oy/HTND/app/protocol/peer"
	"github.com/Hoosat-Oy/HTND/domain"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter/router"
)

// HandleIBDBlockRequestsContext is the interface for the context needed for the HandleIBDBlockRequests flow.
type HandleIBDBlockRequestsContext interface {
	Domain() domain.Domain
}

// HandleIBDBlockRequests listens to appmessage.MsgRequestRelayBlocks messages and sends
// their corresponding blocks to the requesting peer.
func HandleIBDBlockRequests(context HandleIBDBlockRequestsContext, incomingRoute *router.Route,
	outgoingRoute *router.Route, peer *peerpkg.Peer) error {

	for {
		message, err := incomingRoute.Dequeue()
		if err != nil {
			return err
		}
		msgRequestIBDBlocks := message.(*appmessage.MsgRequestIBDBlocks)
		log.Debugf("Got request for %d ibd blocks", len(msgRequestIBDBlocks.Hashes))

		for i := 0; i < len(msgRequestIBDBlocks.Hashes); i++ {
			hash := msgRequestIBDBlocks.Hashes[i]
			go func(hash *externalapi.DomainHash) {
				// Fetch the block from the database.
				block, found, err := context.Domain().Consensus().GetBlock(msgRequestIBDBlocks.Hashes[i])
				if err != nil {
					log.Warnf("unable to fetch requested block hash %s: %s", hash, err)
					return
				}
				if !found {
					log.Warnf("Relay block %s not found", hash)
					return
				}

				// TODO (Partial nodes): Convert block to partial block if needed

				log.Infof("Relaying block %s through IBD to peer %s", hash, peer.Address())
				blockMessage := appmessage.DomainBlockToMsgBlock(block)
				ibdBlockMessage := appmessage.NewMsgIBDBlock(blockMessage)
				err = outgoingRoute.Enqueue(ibdBlockMessage)
				if err != nil {
					log.Warnf("failed to enqueue block %s: %s", hash, err)
					return
				}
			}(hash)
		}
	}
}
