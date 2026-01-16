package txscript

import (
	"testing"

	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/consensushashing"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/subnetworks"
)

func TestCheckTemplateVerify_BadHashLength(t *testing.T) {
	t.Parallel()

	tx := &externalapi.DomainTransaction{
		Version: 1,
		Inputs: []*externalapi.DomainTransactionInput{{
			PreviousOutpoint: externalapi.DomainOutpoint{
				TransactionID: *externalapi.NewDomainTransactionIDFromByteArray(&[externalapi.DomainHashSize]byte{0x01}),
				Index:         0,
			},
			Sequence:   1,
			SigOpCount: 0,
		}},
		Outputs: []*externalapi.DomainTransactionOutput{{
			Value: 1,
			ScriptPublicKey: &externalapi.ScriptPublicKey{
				Version: 0,
				Script:  mustParseShortForm("1", 0),
			},
		}},
		SubnetworkID: subnetworks.SubnetworkIDNative,
	}

	script, err := NewScriptBuilder().
		AddData(make([]byte, externalapi.DomainHashSize-1)).
		AddOp(OpCheckTemplateVerify).
		AddOp(OpTrue).
		Script()
	if err != nil {
		t.Fatalf("script builder: %v", err)
	}

	vm, err := NewEngine(&externalapi.ScriptPublicKey{Script: script, Version: 2}, tx, 0, 0, nil, nil, &consensushashing.SighashReusedValues{})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	err = vm.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !IsErrorCode(err, ErrCheckTemplateVerify) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckTemplateVerify_TxIdxSensitivity(t *testing.T) {
	t.Parallel()

	tx := &externalapi.DomainTransaction{
		Version: 1,
		Inputs: []*externalapi.DomainTransactionInput{{
			PreviousOutpoint: externalapi.DomainOutpoint{
				TransactionID: *externalapi.NewDomainTransactionIDFromByteArray(&[externalapi.DomainHashSize]byte{0x01}),
				Index:         0,
			},
			Sequence:   111,
			SigOpCount: 0,
		}, {
			PreviousOutpoint: externalapi.DomainOutpoint{
				TransactionID: *externalapi.NewDomainTransactionIDFromByteArray(&[externalapi.DomainHashSize]byte{0x02}),
				Index:         1,
			},
			Sequence:   222,
			SigOpCount: 0,
		}},
		Outputs: []*externalapi.DomainTransactionOutput{{
			Value: 7,
			ScriptPublicKey: &externalapi.ScriptPublicKey{
				Version: 0,
				Script:  mustParseShortForm("1", 0),
			},
		}},
		SubnetworkID: subnetworks.SubnetworkIDNative,
	}

	// Template hash is intentionally for input index 1.
	templateHash := calculateTemplateHash(tx, 1)
	script, err := NewScriptBuilder().
		AddData(templateHash.ByteSlice()).
		AddOp(OpCheckTemplateVerify).
		AddOp(OpTrue).
		Script()
	if err != nil {
		t.Fatalf("script builder: %v", err)
	}

	// Verify succeeds when the engine is created with txIdx=1.
	vm, err := NewEngine(&externalapi.ScriptPublicKey{Script: script, Version: 2}, tx, 1, 0, nil, nil, &consensushashing.SighashReusedValues{})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	if err := vm.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// And fails when executed for txIdx=0.
	vm, err = NewEngine(&externalapi.ScriptPublicKey{Script: script, Version: 2}, tx, 0, 0, nil, nil, &consensushashing.SighashReusedValues{})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	err = vm.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !IsErrorCode(err, ErrCheckTemplateVerify) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckTemplateVerify_SubnetworkPayloadAffectsHash(t *testing.T) {
	t.Parallel()

	makeTx := func(payload []byte) *externalapi.DomainTransaction {
		return &externalapi.DomainTransaction{
			Version: 1,
			Inputs: []*externalapi.DomainTransactionInput{{
				PreviousOutpoint: externalapi.DomainOutpoint{
					TransactionID: *externalapi.NewDomainTransactionIDFromByteArray(&[externalapi.DomainHashSize]byte{0x01}),
					Index:         0,
				},
				Sequence:   1,
				SigOpCount: 0,
			}},
			Outputs: []*externalapi.DomainTransactionOutput{{
				Value: 1,
				ScriptPublicKey: &externalapi.ScriptPublicKey{
					Version: 0,
					Script:  mustParseShortForm("1", 0),
				},
			}},
			LockTime:     0,
			SubnetworkID: subnetworks.SubnetworkIDData,
			Gas:          0,
			Payload:      payload,
		}
	}

	tx := makeTx([]byte("payload-a"))
	templateHash := calculateTemplateHash(tx, 0)

	// Mutate the payload to force mismatch.
	tx.Payload = []byte("payload-b")

	script, err := NewScriptBuilder().
		AddData(templateHash.ByteSlice()).
		AddOp(OpCheckTemplateVerify).
		AddOp(OpTrue).
		Script()
	if err != nil {
		t.Fatalf("script builder: %v", err)
	}

	vm, err := NewEngine(&externalapi.ScriptPublicKey{Script: script, Version: 2}, tx, 0, 0, nil, nil, &consensushashing.SighashReusedValues{})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	err = vm.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !IsErrorCode(err, ErrCheckTemplateVerify) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckTemplateVerify_NativePayloadIgnored(t *testing.T) {
	t.Parallel()

	tx := &externalapi.DomainTransaction{
		Version: 1,
		Inputs: []*externalapi.DomainTransactionInput{{
			PreviousOutpoint: externalapi.DomainOutpoint{
				TransactionID: *externalapi.NewDomainTransactionIDFromByteArray(&[externalapi.DomainHashSize]byte{0x01}),
				Index:         0,
			},
			Sequence:   1,
			SigOpCount: 0,
		}},
		Outputs: []*externalapi.DomainTransactionOutput{{
			Value: 1,
			ScriptPublicKey: &externalapi.ScriptPublicKey{
				Version: 0,
				Script:  mustParseShortForm("1", 0),
			},
		}},
		SubnetworkID: subnetworks.SubnetworkIDNative,
		Payload:      []byte("ignored"),
	}

	// For native, payload is hashed as zero-hash.
	templateHash := calculateTemplateHash(tx, 0)

	// Changing payload should not change hash for native.
	tx.Payload = []byte("also ignored")

	script, err := NewScriptBuilder().
		AddData(templateHash.ByteSlice()).
		AddOp(OpCheckTemplateVerify).
		AddOp(OpTrue).
		Script()
	if err != nil {
		t.Fatalf("script builder: %v", err)
	}

	vm, err := NewEngine(&externalapi.ScriptPublicKey{Script: script, Version: 2}, tx, 0, 0, nil, nil, &consensushashing.SighashReusedValues{})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	if err := vm.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckTemplateVerify_ConsumesStackElement(t *testing.T) {
	t.Parallel()

	tx := &externalapi.DomainTransaction{
		Version: 1,
		Inputs: []*externalapi.DomainTransactionInput{{
			PreviousOutpoint: externalapi.DomainOutpoint{
				TransactionID: *externalapi.NewDomainTransactionIDFromByteArray(&[externalapi.DomainHashSize]byte{0x01}),
				Index:         0,
			},
			Sequence:   1,
			SigOpCount: 0,
		}},
		Outputs: []*externalapi.DomainTransactionOutput{{
			Value: 1,
			ScriptPublicKey: &externalapi.ScriptPublicKey{
				Version: 0,
				Script:  mustParseShortForm("1", 0),
			},
		}},
		SubnetworkID: subnetworks.SubnetworkIDNative,
	}

	templateHash := calculateTemplateHash(tx, 0)

	script, err := NewScriptBuilder().
		AddOp(OpTrue). // sentinel
		AddData(templateHash.ByteSlice()).
		AddOp(OpCheckTemplateVerify).
		AddOp(OpDrop). // keep script clean-stack valid
		AddOp(OpTrue).
		Script()
	if err != nil {
		t.Fatalf("script builder: %v", err)
	}

	// Step through and assert that OP_CHECKTEMPLATEVERIFY consumes exactly one
	// element (the template hash), leaving the sentinel behind.
	vm, err := NewEngine(&externalapi.ScriptPublicKey{Script: script, Version: 2}, tx, 0, 0, nil, nil, &consensushashing.SighashReusedValues{})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	// OP_TRUE
	if _, err := vm.Step(); err != nil {
		t.Fatalf("Step: %v", err)
	}
	if got := len(vm.GetStack()); got != 1 {
		t.Fatalf("unexpected stack size after OP_TRUE: %d", got)
	}

	// pushdata(templateHash)
	if _, err := vm.Step(); err != nil {
		t.Fatalf("Step: %v", err)
	}
	if got := len(vm.GetStack()); got != 2 {
		t.Fatalf("unexpected stack size after pushing hash: %d", got)
	}

	// OP_CHECKTEMPLATEVERIFY (consumes hash)
	if _, err := vm.Step(); err != nil {
		t.Fatalf("Step: %v", err)
	}
	if got := len(vm.GetStack()); got != 1 {
		t.Fatalf("unexpected stack size after OP_CHECKTEMPLATEVERIFY: %d", got)
	}

	// Execute the rest of the script to ensure the full program is valid.
	for {
		done, err := vm.Step()
		if err != nil {
			t.Fatalf("Step: %v", err)
		}
		if done {
			break
		}
	}
	if err := vm.CheckErrorCondition(true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
