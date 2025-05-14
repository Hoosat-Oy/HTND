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
		}
	}
	// for {
	// 	message, err := incomingRoute.Dequeue()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	getRelayBlocksMessage := message.(*appmessage.MsgRequestRelayBlocks)

	// 	var wg sync.WaitGroup
	// 	errChan := make(chan error, len(getRelayBlocksMessage.Hashes))

	// 	for i := 0; i < len(getRelayBlocksMessage.Hashes); i++ {
	// 		hash := getRelayBlocksMessage.Hashes[i]

	// 		wg.Add(1)
	// 		go func(hash *externalapi.DomainHash) {
	// 			defer wg.Done()

	// 			block, found, err := context.Domain().Consensus().GetBlock(hash)
	// 			if err != nil {
	// 				errChan <- errors.Wrapf(err, "unable to fetch requested block hash %s", hash)
	// 				return
	// 			}
	// 			if !found {
	// 				errChan <- protocolerrors.Errorf(false, "Relay block %s not found", hash)
	// 				return
	// 			}

	// 			if block.PoWHash == "" && block.Header.Version() >= constants.PoWIntegrityMinVersion {
	// 				powHashFound := false
	// 				for j := 0; j < 5; j++ {
	// 					time.Sleep(30 * time.Millisecond)
	// 					block, found, err = context.Domain().Consensus().GetBlock(hash)
	// 					if err != nil {
	// 						errChan <- errors.Wrapf(err, "unable to re-fetch requested block hash %s", hash)
	// 						return
	// 					}
	// 					if !found {
	// 						errChan <- protocolerrors.Errorf(false, "Relay block %s not found on retry", hash)
	// 						return
	// 					}
	// 					if block.PoWHash != "" {
	// 						powHashFound = true
	// 						break
	// 					}
	// 				}
	// 				if !powHashFound {
	// 					state := pow.NewState(block.Header.ToMutable())
	// 					_, powHash := state.CalculateProofOfWorkValue()
	// 					block.PoWHash = powHash.String()
	// 				}
	// 			}

	// 			err = outgoingRoute.Enqueue(appmessage.DomainBlockToMsgBlock(block))
	// 			if err != nil {
	// 				errChan <- errors.Wrapf(err, "failed to enqueue block %s", hash)
	// 				return
	// 			}
	// 		}(hash)
	// 	}

	// 	wg.Wait()
	// 	close(errChan)

	// 	for err := range errChan {
	// 		if err != nil {
	// 			return err
	// 		}
	// 	}
	// }
}
