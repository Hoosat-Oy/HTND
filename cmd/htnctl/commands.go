package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter/server/grpcserver/protowire"
)

var commandTypes = []reflect.Type{
	reflect.TypeOf(protowire.HoosatdMessage_AddPeerRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_GetConnectedPeerInfoRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_GetPeerAddressesRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_GetCurrentNetworkRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_GetInfoRequest{}),

	reflect.TypeOf(protowire.HoosatdMessage_GetBlockRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_GetBlockByTransactionIdRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_GetBlocksRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_GetHeadersRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_GetBlockCountRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_GetBlockDagInfoRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_GetSelectedTipHashRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_GetVirtualSelectedParentBlueScoreRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_GetVirtualSelectedParentChainFromBlockRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_ResolveFinalityConflictRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_EstimateNetworkHashesPerSecondRequest{}),

	reflect.TypeOf(protowire.HoosatdMessage_GetBlockTemplateRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_SubmitBlockRequest{}),

	reflect.TypeOf(protowire.HoosatdMessage_GetMempoolEntryRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_GetMempoolEntriesRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_GetMempoolEntriesByAddressesRequest{}),

	reflect.TypeOf(protowire.HoosatdMessage_SubmitTransactionRequest{}),

	reflect.TypeOf(protowire.HoosatdMessage_GetUtxosByAddressesRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_GetBalanceByAddressRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_GetCoinSupplyRequest{}),

	reflect.TypeOf(protowire.HoosatdMessage_BanRequest{}),
	reflect.TypeOf(protowire.HoosatdMessage_UnbanRequest{}),
}

type commandDescription struct {
	name       string
	parameters []*parameterDescription
	typeof     reflect.Type
}

type parameterDescription struct {
	name   string
	typeof reflect.Type
}

func commandDescriptions() []*commandDescription {
	commandDescriptions := make([]*commandDescription, len(commandTypes))

	for i, commandTypeWrapped := range commandTypes {
		commandType := unwrapCommandType(commandTypeWrapped)

		name := strings.TrimSuffix(commandType.Name(), "RequestMessage")
		numFields := commandType.NumField()

		var parameters []*parameterDescription
		for i := 0; i < numFields; i++ {
			field := commandType.Field(i)

			if !isFieldExported(field) {
				continue
			}

			parameters = append(parameters, &parameterDescription{
				name:   field.Name,
				typeof: field.Type,
			})
		}
		commandDescriptions[i] = &commandDescription{
			name:       name,
			parameters: parameters,
			typeof:     commandTypeWrapped,
		}
	}

	return commandDescriptions
}

func (cd *commandDescription) help() string {
	sb := &strings.Builder{}
	sb.WriteString(cd.name)
	for _, parameter := range cd.parameters {
		_, _ = fmt.Fprintf(sb, " [%s]", parameter.name)
	}
	return sb.String()
}
