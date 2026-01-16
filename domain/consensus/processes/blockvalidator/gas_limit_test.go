package blockvalidator

import (
	"testing"

	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/ruleerrors"
	"github.com/pkg/errors"
)

func TestValidateGasLimit(t *testing.T) {
	v := &blockValidator{maxGasPerSubnetworkPerBlock: 100}

	block := &externalapi.DomainBlock{Transactions: []*externalapi.DomainTransaction{
		{SubnetworkID: externalapi.DomainSubnetworkID{4}, Gas: 60},
		{SubnetworkID: externalapi.DomainSubnetworkID{4}, Gas: 41},
	}}

	err := v.validateGasLimit(block)
	if !errors.Is(err, ruleerrors.ErrInvalidGas) {
		t.Fatalf("expected ErrInvalidGas, got: %+v", err)
	}

	block = &externalapi.DomainBlock{Transactions: []*externalapi.DomainTransaction{
		{SubnetworkID: externalapi.DomainSubnetworkID{4}, Gas: 60},
		{SubnetworkID: externalapi.DomainSubnetworkID{4}, Gas: 40},
		{SubnetworkID: externalapi.DomainSubnetworkID{5}, Gas: 100},
	}}
	err = v.validateGasLimit(block)
	if err != nil {
		t.Fatalf("expected nil, got: %+v", err)
	}
}
