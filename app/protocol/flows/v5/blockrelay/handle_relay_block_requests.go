package blockrelay

import (
	"time"

	"github.com/Hoosat-Oy/HTND/app/appmessage"
	peerpkg "github.com/Hoosat-Oy/HTND/app/protocol/peer"
	"github.com/Hoosat-Oy/HTND/app/protocol/protocolerrors"
	"github.com/Hoosat-Oy/HTND/domain"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/constants"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/pow"
	"github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter/router"
	"github.com/pkg/errors"
)

// RelayBlockRequestsContext is the interface for the context needed for the HandleRelayBlockRequests flow.
type RelayBlockRequestsContext interface {
	Domain() domain.Domain
}

// HandleRelayBlockRequests listens to appmessage.MsgRequestRelayBlocks messages and sends
// their corresponding blocks to the requesting peer.
func HandleRelayBlockRequests(context RelayBlockRequestsContext, incomingRoute *router.Route,
	outgoingRoute *router.Route, peer *peerpkg.Peer) error {

	for {
		message, err := incomingRoute.Dequeue()
		if err != nil {
			return err
		}
		getRelayBlocksMessage := message.(*appmessage.MsgRequestRelayBlocks)
		// log.Debugf("Got request for relay blocks with hashes %s", getRelayBlocksMessage.Hashes)
		for _, hash := range getRelayBlocksMessage.Hashes {
			// Fetch the block from the database.

			block, found, err := context.Domain().Consensus().GetBlock(hash)
			if err != nil {
				return errors.Wrapf(err, "unable to fetch requested block hash %s", hash)
			}
			if !found {
				return protocolerrors.Errorf(false, "Relay block %s not found", hash)
			}

			if block.PoWHash == "" && block.Header.Version() >= constants.PoWIntegrityMinVersion {
				powHashFound := false
				for i := 0; i < 5; i++ {
					time.Sleep(30 * time.Millisecond)
					block, found, err = context.Domain().Consensus().GetBlock(hash)
					if err != nil {
						return errors.Wrapf(err, "unable to fetch requested block hash %s", hash)
					}
					if !found {
						return protocolerrors.Errorf(false, "Relay block %s not found", hash)
					}
					if block.PoWHash != "" {
						// log.Debugf("Found PoW Hash for relayed block: Hash %s, Pow hash %s", hash, block.PoWHash)
						powHashFound = true
						break
					}
				}
				if !powHashFound {
					state := pow.NewState(block.Header.ToMutable())
					_, powHash := state.CalculateProofOfWorkValue()
					block.PoWHash = powHash.String()
					// log.Debugf("Recalculated PoW Hash for relayed block: Hash %s, Pow hash %s", hash, block.PoWHash)
				}
			}

			err = outgoingRoute.Enqueue(appmessage.DomainBlockToMsgBlock(block))
			if err != nil {
				return err
			}
			// log.Debugf("Relayed block: Hash %s, Pow hash %s, Version %d >= %d, Peer %s", hash, block.PoWHash, block.Header.Version(), constants.PoWIntegrityMinVersion, peer.Connection().Address())
		}
	}
}
