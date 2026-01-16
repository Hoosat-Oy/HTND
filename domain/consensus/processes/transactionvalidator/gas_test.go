package transactionvalidator

import (
	"math"
	"testing"

	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/ruleerrors"
	"github.com/pkg/errors"
)

func TestCheckGasLimitInNonBuiltInSubnetworkTransaction(t *testing.T) {
	v := &transactionValidator{maxGasPerSubnetworkPerBlock: 100}

	tx := &externalapi.DomainTransaction{SubnetworkID: externalapi.DomainSubnetworkID{4}, Gas: 101}
	err := v.checkGasLimitInNonBuiltInSubnetworkTransaction(tx)
	if !errors.Is(err, ruleerrors.ErrInvalidGas) {
		t.Fatalf("expected ErrInvalidGas, got: %+v", err)
	}

	tx.Gas = 100
	err = v.checkGasLimitInNonBuiltInSubnetworkTransaction(tx)
	if err != nil {
		t.Fatalf("expected nil, got: %+v", err)
	}
}

func TestCheckMinFeePerGas(t *testing.T) {
	v := &transactionValidator{minFeePerGas: 2}
	tx := &externalapi.DomainTransaction{SubnetworkID: externalapi.DomainSubnetworkID{4}, Gas: 10, Fee: 19}

	err := v.checkMinFeePerGas(tx)
	if !errors.Is(err, ruleerrors.ErrInsufficientGasFee) {
		t.Fatalf("expected ErrInsufficientGasFee, got: %+v", err)
	}

	tx.Fee = 20
	err = v.checkMinFeePerGas(tx)
	if err != nil {
		t.Fatalf("expected nil, got: %+v", err)
	}

	// Overflow case should fail.
	v = &transactionValidator{minFeePerGas: 2}
	tx = &externalapi.DomainTransaction{SubnetworkID: externalapi.DomainSubnetworkID{4}, Gas: math.MaxUint64, Fee: math.MaxUint64}
	err = v.checkMinFeePerGas(tx)
	if !errors.Is(err, ruleerrors.ErrInsufficientGasFee) {
		t.Fatalf("expected ErrInsufficientGasFee on overflow, got: %+v", err)
	}
}
