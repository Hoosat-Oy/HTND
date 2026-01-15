// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"testing"

	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/consensushashing"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/constants"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/subnetworks"

	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
)

func TestSpliceOpcodesEnabledInV1(t *testing.T) {
	t.Parallel()

	inputs := []*externalapi.DomainTransactionInput{{
		PreviousOutpoint: externalapi.DomainOutpoint{
			TransactionID: *externalapi.NewDomainTransactionIDFromByteArray(&[externalapi.DomainHashSize]byte{}),
			Index:         0,
		},
		SignatureScript: nil,
		Sequence:        constants.MaxTxInSequenceNum,
	}}
	outputs := []*externalapi.DomainTransactionOutput{{
		Value:           0,
		ScriptPublicKey: nil,
	}}
	tx := &externalapi.DomainTransaction{
		Version: 1,
		Inputs:  inputs,
		Outputs: outputs,
	}

	tests := []struct {
		name        string
		version     uint16
		script      string
		expectError bool
		errCode     ErrorCode
	}{
		{
			name:        "v0 disabled opcode fails even when unexecuted",
			version:     0,
			script:      "0 IF CAT ENDIF 1",
			expectError: true,
			errCode:     ErrDisabledOpcode,
		},
		{
			name:        "v1 allows opcode in unexecuted branch",
			version:     1,
			script:      "0 IF CAT ENDIF 1",
			expectError: false,
		},
		{
			name:        "v1 OP_CAT executes",
			version:     1,
			script:      "'ab' 'cd' CAT 'abcd' EQUAL",
			expectError: false,
		},
		{
			name:        "v1 OP_LEFT executes",
			version:     1,
			script:      "'abcd' 2 LEFT 'ab' EQUAL",
			expectError: false,
		},
		{
			name:        "v1 OP_RIGHT executes",
			version:     1,
			script:      "'abcd' 2 RIGHT 'cd' EQUAL",
			expectError: false,
		},
		{
			name:        "v1 OP_SUBSTR executes",
			version:     1,
			script:      "'abcdef' 2 3 SUBSTR 'cde' EQUAL",
			expectError: false,
		},
		{
			name:        "v0 splice opcode remains disabled",
			version:     0,
			script:      "'ab' 'cd' CAT 'abcd' EQUAL",
			expectError: true,
			errCode:     ErrDisabledOpcode,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			scriptPubKey := &externalapi.ScriptPublicKey{Script: mustParseShortForm(test.script, test.version), Version: test.version}
			vm, err := NewEngine(scriptPubKey, tx, 0, 0, nil, nil, &consensushashing.SighashReusedValues{})
			if err != nil {
				t.Fatalf("NewEngine: %v", err)
			}
			err = vm.Execute()
			if test.expectError {
				if err == nil {
					t.Fatalf("expected error %v but got nil", test.errCode)
				}
				if !IsErrorCode(err, test.errCode) {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCheckTemplateVerify(t *testing.T) {
	t.Parallel()

	makeTx := func() *externalapi.DomainTransaction {
		inputs := []*externalapi.DomainTransactionInput{{
			PreviousOutpoint: externalapi.DomainOutpoint{
				TransactionID: *externalapi.NewDomainTransactionIDFromByteArray(&[externalapi.DomainHashSize]byte{0x01}),
				Index:         7,
			},
			SignatureScript: nil,
			Sequence:        123,
			SigOpCount:      0,
		}}
		outputs := []*externalapi.DomainTransactionOutput{{
			Value: 42,
			ScriptPublicKey: &externalapi.ScriptPublicKey{
				Version: 0,
				Script:  mustParseShortForm("1", 0),
			},
		}}
		return &externalapi.DomainTransaction{
			Version:      1,
			Inputs:       inputs,
			Outputs:      outputs,
			LockTime:     0,
			SubnetworkID: subnetworks.SubnetworkIDNative,
			Gas:          0,
			Payload:      nil,
		}
	}

	t.Run("v2 matches template", func(t *testing.T) {
		tx := makeTx()
		templateHash := calculateTemplateHash(tx, 0)

		script, err := NewScriptBuilder().
			AddData(templateHash.ByteSlice()).
			AddOp(OpCheckTemplateVerify).
			AddOp(OpTrue).
			Script()
		if err != nil {
			t.Fatalf("script builder: %v", err)
		}
		scriptPubKey := &externalapi.ScriptPublicKey{Script: script, Version: 2}

		vm, err := NewEngine(scriptPubKey, tx, 0, 0, nil, nil, &consensushashing.SighashReusedValues{})
		if err != nil {
			t.Fatalf("NewEngine: %v", err)
		}
		if err := vm.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("v2 template mismatch fails", func(t *testing.T) {
		tx := makeTx()
		templateHash := calculateTemplateHash(tx, 0)

		// Mutate tx after producing hash to force mismatch.
		tx.Outputs[0].Value = 43

		script, err := NewScriptBuilder().
			AddData(templateHash.ByteSlice()).
			AddOp(OpCheckTemplateVerify).
			AddOp(OpTrue).
			Script()
		if err != nil {
			t.Fatalf("script builder: %v", err)
		}
		scriptPubKey := &externalapi.ScriptPublicKey{Script: script, Version: 2}

		vm, err := NewEngine(scriptPubKey, tx, 0, 0, nil, nil, &consensushashing.SighashReusedValues{})
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
	})

	t.Run("v1 rejects opcode", func(t *testing.T) {
		tx := makeTx()
		templateHash := calculateTemplateHash(tx, 0)

		script, err := NewScriptBuilder().
			AddData(templateHash.ByteSlice()).
			AddOp(OpCheckTemplateVerify).
			AddOp(OpTrue).
			Script()
		if err != nil {
			t.Fatalf("script builder: %v", err)
		}
		scriptPubKey := &externalapi.ScriptPublicKey{Script: script, Version: 1}

		vm, err := NewEngine(scriptPubKey, tx, 0, 0, nil, nil, &consensushashing.SighashReusedValues{})
		if err != nil {
			t.Fatalf("NewEngine: %v", err)
		}
		err = vm.Execute()
		if err == nil {
			t.Fatalf("expected error")
		}
		if !IsErrorCode(err, ErrReservedOpcode) {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestBadPC sets the pc to a deliberately bad result then confirms that Step()
// and Disasm fail correctly.
func TestBadPC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		script, off int
	}{
		{script: 2, off: 0},
		{script: 0, off: 2},
	}

	// tx with almost empty scripts.
	inputs := []*externalapi.DomainTransactionInput{
		{
			PreviousOutpoint: externalapi.DomainOutpoint{
				TransactionID: *externalapi.NewDomainTransactionIDFromByteArray(&[externalapi.DomainHashSize]byte{
					0xc9, 0x97, 0xa5, 0xe5,
					0x6e, 0x10, 0x41, 0x02,
					0xfa, 0x20, 0x9c, 0x6a,
					0x85, 0x2d, 0xd9, 0x06,
					0x60, 0xa2, 0x0b, 0x2d,
					0x9c, 0x35, 0x24, 0x23,
					0xed, 0xce, 0x25, 0x85,
					0x7f, 0xcd, 0x37, 0x04,
				}),
				Index: 0,
			},
			SignatureScript: mustParseShortForm("", 0),
			Sequence:        4294967295,
		},
	}
	outputs := []*externalapi.DomainTransactionOutput{{
		Value:           1000000000,
		ScriptPublicKey: nil,
	}}
	tx := &externalapi.DomainTransaction{
		Version: 1,
		Inputs:  inputs,
		Outputs: outputs,
	}
	scriptPubKey := &externalapi.ScriptPublicKey{Script: mustParseShortForm("NOP", 0), Version: 0}

	for _, test := range tests {
		vm, err := NewEngine(scriptPubKey, tx, 0, 0, nil, nil, &consensushashing.SighashReusedValues{})
		if err != nil {
			t.Errorf("Failed to create script: %v", err)
		}

		// set to after all scripts
		vm.scriptIdx = test.script
		vm.scriptOff = test.off

		_, err = vm.Step()
		if err == nil {
			t.Errorf("Step with invalid pc (%v) succeeds!", test)
			continue
		}

		_, err = vm.DisasmPC()
		if err == nil {
			t.Errorf("DisasmPC with invalid pc (%v) succeeds!",
				test)
		}
	}
}

func TestCheckErrorCondition(t *testing.T) {
	tests := []struct {
		script      string
		finalScript bool
		stepCount   int
		expectedErr error
	}{
		{"OP_1", true, 1, nil},
		{"NOP", true, 0, scriptError(ErrScriptUnfinished, "")},
		{"NOP", true, 1, scriptError(ErrEmptyStack, "")},
		{"OP_1 OP_1", true, 2, scriptError(ErrCleanStack, "")},
		{"OP_0", true, 1, scriptError(ErrEvalFalse, "")},
	}

	for i, test := range tests {
		func() {
			inputs := []*externalapi.DomainTransactionInput{{
				PreviousOutpoint: externalapi.DomainOutpoint{
					TransactionID: *externalapi.NewDomainTransactionIDFromByteArray(&[externalapi.DomainHashSize]byte{
						0xc9, 0x97, 0xa5, 0xe5,
						0x6e, 0x10, 0x41, 0x02,
						0xfa, 0x20, 0x9c, 0x6a,
						0x85, 0x2d, 0xd9, 0x06,
						0x60, 0xa2, 0x0b, 0x2d,
						0x9c, 0x35, 0x24, 0x23,
						0xed, 0xce, 0x25, 0x85,
						0x7f, 0xcd, 0x37, 0x04,
					}),
					Index: 0,
				},
				SignatureScript: nil,
				Sequence:        4294967295,
			}}
			outputs := []*externalapi.DomainTransactionOutput{{
				Value:           1000000000,
				ScriptPublicKey: nil,
			}}
			tx := &externalapi.DomainTransaction{
				Version: 1,
				Inputs:  inputs,
				Outputs: outputs,
			}

			scriptPubKey := &externalapi.ScriptPublicKey{Script: mustParseShortForm(test.script, 0), Version: 0}

			vm, err := NewEngine(scriptPubKey, tx, 0, 0, nil, nil, &consensushashing.SighashReusedValues{})
			if err != nil {
				t.Errorf("TestCheckErrorCondition: %d: failed to create script: %v", i, err)
			}

			for j := 0; j < test.stepCount; j++ {
				_, err = vm.Step()
				if err != nil {
					t.Errorf("TestCheckErrorCondition: %d: failed to execute step No. %d: %v", i, j+1, err)
					return
				}

				if j != test.stepCount-1 {
					err = vm.CheckErrorCondition(false)
					if !IsErrorCode(err, ErrScriptUnfinished) {
						t.Fatalf("TestCheckErrorCondition: %d: got unexepected error %v on %dth iteration",
							i, err, j)
						return
					}
				}
			}

			err = vm.CheckErrorCondition(test.finalScript)
			if e := checkScriptError(err, test.expectedErr); e != nil {
				t.Errorf("TestCheckErrorCondition: %d: %s", i, e)
			}
		}()
	}
}

// TestCheckPubKeyEncoding ensures the internal checkPubKeyEncoding function
// works as expected.
func TestCheckPubKeyEncoding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     []byte
		isValid bool
	}{
		{
			name: "uncompressed - invalid",
			key: hexToBytes("0411db93e1dcdb8a016b49840f8c53bc1eb68" +
				"a382e97b1482ecad7b148a6909a5cb2e0eaddfb84ccf" +
				"9744464f82e160bfa9b8b64f9d4c03f999b8643f656b" +
				"412a3"),
			isValid: false,
		},
		{
			name: "compressed - invalid",
			key: hexToBytes("02ce0b14fb842b1ba549fdd675c98075f12e9" +
				"c510f8ef52bd021a9a1f4809d3b4d"),
			isValid: false,
		},
		{
			name: "compressed - invalid",
			key: hexToBytes("032689c7c2dab13309fb143e0e8fe39634252" +
				"1887e976690b6b47f5b2a4b7d448e"),
			isValid: false,
		},
		{
			name: "hybrid - invalid",
			key: hexToBytes("0679be667ef9dcbbac55a06295ce870b07029" +
				"bfcdb2dce28d959f2815b16f81798483ada7726a3c46" +
				"55da4fbfc0e1108a8fd17b448a68554199c47d08ffb1" +
				"0d4b8"),
			isValid: false,
		},
		{
			name:    "32 bytes pubkey - Ok",
			key:     hexToBytes("2689c7c2dab13309fb143e0e8fe396342521887e976690b6b47f5b2a4b7d448e"),
			isValid: true,
		},
		{
			name:    "empty",
			key:     nil,
			isValid: false,
		},
	}

	vm := Engine{}
	for _, test := range tests {
		err := vm.checkPubKeyEncoding(test.key)
		if err != nil && test.isValid {
			t.Errorf("checkSignatureLength test '%s' failed "+
				"when it should have succeeded: %v", test.name,
				err)
		} else if err == nil && !test.isValid {
			t.Errorf("checkSignatureEncooding test '%s' succeeded "+
				"when it should have failed", test.name)
		}
	}

}

func TestDisasmPC(t *testing.T) {
	t.Parallel()

	// tx with almost empty scripts.
	inputs := []*externalapi.DomainTransactionInput{{
		PreviousOutpoint: externalapi.DomainOutpoint{
			TransactionID: *externalapi.NewDomainTransactionIDFromByteArray(&[externalapi.DomainHashSize]byte{
				0xc9, 0x97, 0xa5, 0xe5,
				0x6e, 0x10, 0x41, 0x02,
				0xfa, 0x20, 0x9c, 0x6a,
				0x85, 0x2d, 0xd9, 0x06,
				0x60, 0xa2, 0x0b, 0x2d,
				0x9c, 0x35, 0x24, 0x23,
				0xed, 0xce, 0x25, 0x85,
				0x7f, 0xcd, 0x37, 0x04,
			}),
			Index: 0,
		},
		SignatureScript: mustParseShortForm("OP_2", 0),
		Sequence:        4294967295,
	}}
	outputs := []*externalapi.DomainTransactionOutput{{
		Value:           1000000000,
		ScriptPublicKey: nil,
	}}
	tx := &externalapi.DomainTransaction{
		Version: 1,
		Inputs:  inputs,
		Outputs: outputs,
	}

	scriptPubKey := &externalapi.ScriptPublicKey{Script: mustParseShortForm("OP_DROP NOP TRUE", 0), Version: 0}

	vm, err := NewEngine(scriptPubKey, tx, 0, 0, nil, nil, &consensushashing.SighashReusedValues{})
	if err != nil {
		t.Fatalf("failed to create script: %v", err)
	}

	tests := []struct {
		expected    string
		expectedErr error
	}{
		{"00:0000: OP_2", nil},
		{"01:0000: OP_DROP", nil},
		{"01:0001: OP_NOP", nil},
		{"01:0002: OP_1", nil},
		{"", scriptError(ErrInvalidProgramCounter, "")},
	}

	for i, test := range tests {
		actual, err := vm.DisasmPC()
		if e := checkScriptError(err, test.expectedErr); e != nil {
			t.Errorf("TestDisasmPC: %d: %s", i, e)
		}

		if actual != test.expected {
			t.Errorf("TestDisasmPC: %d: expected: '%s'. Got: '%s'", i, test.expected, actual)
		}

		// ignore results from vm.Step() to keep going even when no opcodes left, to hit error case
		_, _ = vm.Step()
	}
}

func TestDisasmScript(t *testing.T) {
	t.Parallel()

	// tx with almost empty scripts.
	inputs := []*externalapi.DomainTransactionInput{{
		PreviousOutpoint: externalapi.DomainOutpoint{
			TransactionID: *externalapi.NewDomainTransactionIDFromByteArray(&[externalapi.DomainHashSize]byte{
				0xc9, 0x97, 0xa5, 0xe5,
				0x6e, 0x10, 0x41, 0x02,
				0xfa, 0x20, 0x9c, 0x6a,
				0x85, 0x2d, 0xd9, 0x06,
				0x60, 0xa2, 0x0b, 0x2d,
				0x9c, 0x35, 0x24, 0x23,
				0xed, 0xce, 0x25, 0x85,
				0x7f, 0xcd, 0x37, 0x04,
			}),
			Index: 0,
		},
		SignatureScript: mustParseShortForm("OP_2", 0),
		Sequence:        4294967295,
	}}
	outputs := []*externalapi.DomainTransactionOutput{{
		Value:           1000000000,
		ScriptPublicKey: nil,
	}}
	tx := &externalapi.DomainTransaction{
		Version: 1,
		Inputs:  inputs,
		Outputs: outputs,
	}

	scriptPubKey := &externalapi.ScriptPublicKey{Script: mustParseShortForm("OP_DROP NOP TRUE", 0), Version: 0}
	vm, err := NewEngine(scriptPubKey, tx, 0, 0, nil, nil, &consensushashing.SighashReusedValues{})
	if err != nil {
		t.Fatalf("failed to create script: %v", err)
	}

	tests := []struct {
		index       int
		expected    string
		expectedErr error
	}{
		{-1, "", scriptError(ErrInvalidIndex, "")},
		{0, "00:0000: OP_2\n", nil},
		{1, "01:0000: OP_DROP\n01:0001: OP_NOP\n01:0002: OP_1\n", nil},
		{2, "", scriptError(ErrInvalidIndex, "")},
	}

	for _, test := range tests {
		actual, err := vm.DisasmScript(test.index)
		if e := checkScriptError(err, test.expectedErr); e != nil {
			t.Errorf("TestDisasmScript: %d: %s", test.index, e)
		}

		if actual != test.expected {
			t.Errorf("TestDisasmScript: %d: expected: '%s'. Got: '%s'", test.index, test.expected, actual)
		}
	}
}
