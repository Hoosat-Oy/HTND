package rpchandlers

import (
	"runtime/debug"

	"github.com/Hoosat-Oy/HTND/app/appmessage"
	"github.com/Hoosat-Oy/HTND/app/rpc/rpccontext"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/transactionhelper"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/txscript"
	"github.com/Hoosat-Oy/HTND/infrastructure/db/database"
	"github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter/router"
	"github.com/Hoosat-Oy/HTND/util"
	"github.com/Hoosat-Oy/HTND/version"
)

// HandleGetBlockTemplate handles the respectively named RPC command
func HandleGetBlockTemplate(context *rpccontext.Context, _ *router.Router, request appmessage.Message) (response appmessage.Message, err error) {
	// only convert specific not-found panics to an RPC error with backtrace; rethrow others
	type notFoundPanic struct{ cause error }

	defer func() {
		if r := recover(); r != nil {
			if nf, ok := r.(notFoundPanic); ok {
				stack := debug.Stack()
				log.Warnf("Backtrace:\n%s", stack)
				errorMessage := &appmessage.GetBlockTemplateResponseMessage{}
				errorMessage.Error = appmessage.RPCErrorf("GetBlockTemplate failed with database not found error: %v\n", nf.cause)
				response = errorMessage
				err = nil
				return
			}
			panic(r)
		}
	}()
	getBlockTemplateRequest := request.(*appmessage.GetBlockTemplateRequestMessage)

	payAddress, err := util.DecodeAddress(getBlockTemplateRequest.PayAddress, context.Config.ActiveNetParams.Prefix)
	if err != nil {
		errorMessage := &appmessage.GetBlockTemplateResponseMessage{}
		errorMessage.Error = appmessage.RPCErrorf("Could not decode address: %s", err)
		return errorMessage, nil
	}

	scriptPublicKey, err := txscript.PayToAddrScript(payAddress)
	if err != nil {
		return nil, err
	}

	coinbaseData := &externalapi.DomainCoinbaseData{ScriptPublicKey: scriptPublicKey, ExtraData: []byte(version.Version() + "/" + getBlockTemplateRequest.ExtraData)}

	templateBlock, isNearlySynced, err := context.Domain.MiningManager().GetBlockTemplate(coinbaseData)
	if err != nil {
		if database.IsNotFoundError(err) {
			// Panic and let the deferred recover above send a backtrace to the caller
			panic(notFoundPanic{cause: err})
		}
		return nil, err
	}

	if uint64(len(templateBlock.Transactions[transactionhelper.CoinbaseTransactionIndex].Payload)) > context.Config.NetParams().MaxCoinbasePayloadLength {

	}

	rpcBlock := appmessage.DomainBlockToRPCBlock(templateBlock)

	return appmessage.NewGetBlockTemplateResponseMessage(rpcBlock, context.ProtocolManager.Context().HasPeers() && isNearlySynced), nil
}
