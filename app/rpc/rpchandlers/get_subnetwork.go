package rpchandlers

import (
	"github.com/Hoosat-Oy/HTND/app/appmessage"
	"github.com/Hoosat-Oy/HTND/app/rpc/rpccontext"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/subnetworks"
	"github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter/router"
)

// HandleGetSubnetwork handles the respectively named RPC command
func HandleGetSubnetwork(context *rpccontext.Context, _ *router.Router, request appmessage.Message) (appmessage.Message, error) {
	getSubnetworkRequest := request.(*appmessage.GetSubnetworkRequestMessage)
	response := &appmessage.GetSubnetworkResponseMessage{}

	subnetworkID, err := subnetworks.FromString(getSubnetworkRequest.SubnetworkID)
	if err != nil {
		response.Error = appmessage.RPCErrorf("invalid subnetwork id: %s", err)
		return response, nil
	}

	if subnetworks.IsBuiltInOrNative(*subnetworkID) {
		response.GasLimit = 0
		return response, nil
	}

	response.GasLimit = context.Config.ActiveNetParams.MaxGasPerSubnetworkPerBlock
	return response, nil
}
