package transactionvalidator

import (
	"bytes"
	"encoding/ascii85"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"regexp"
	"strings"

	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/ruleerrors"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/constants"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/subnetworks"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/transactionhelper"
	"github.com/pkg/errors"
)

// ValidateTransactionInIsolation validates the parts of the transaction that can be validated context-free
func (v *transactionValidator) ValidateTransactionInIsolation(tx *externalapi.DomainTransaction, povDAAScore uint64) error {
	err := v.checkTransactionInputCount(tx)
	if err != nil {
		return err
	}
	err = v.checkTransactionAmountRanges(tx)
	if err != nil {
		return err
	}
	err = v.checkDuplicateTransactionInputs(tx)
	if err != nil {
		return err
	}
	err = v.checkCoinbaseInIsolation(tx)
	if err != nil {
		return err
	}
	err = v.checkGasInBuiltInOrNativeTransactions(tx)
	if err != nil {
		return err
	}
	err = v.checkSubnetworkRegistryTransaction(tx)
	if err != nil {
		return err
	}

	err = v.checkDataTransactionPayload(tx)
	if err != nil {
		return err
	}

	err = v.checkNativeTransactionPayload(tx)
	if err != nil {
		return err
	}

	// TODO: fill it with the node's subnetwork id.
	err = v.checkTransactionSubnetwork(tx, nil)
	if err != nil {
		return err
	}

	if tx.Version > constants.MaxTransactionVersion {
		return errors.Wrapf(ruleerrors.ErrTransactionVersionIsUnknown, "validation failed: unknown transaction version. ")
	}

	return nil
}

func (v *transactionValidator) checkTransactionInputCount(tx *externalapi.DomainTransaction) error {
	// A non-coinbase transaction must have at least one input.
	if !transactionhelper.IsCoinBase(tx) && len(tx.Inputs) == 0 {
		return errors.Wrapf(ruleerrors.ErrNoTxInputs, "transaction has no inputs")
	}
	return nil
}

func (v *transactionValidator) checkTransactionAmountRanges(tx *externalapi.DomainTransaction) error {
	// Ensure the transaction amounts are in range. Each transaction
	// output must not be negative or more than the max allowed per
	// transaction. Also, the total of all outputs must abide by the same
	// restrictions. All amounts in a transaction are in a unit value known
	// as a sompi. One hoosat is a quantity of sompi as defined by the
	// sompiPerHoosat constant.
	var totalSompi uint64
	for _, txOut := range tx.Outputs {
		sompi := txOut.Value
		if sompi == 0 {
			return errors.Wrap(ruleerrors.ErrTxOutValueZero, "zero value outputs are forbidden")
		}

		if sompi > constants.MaxSompi {
			return errors.Wrapf(ruleerrors.ErrBadTxOutValue, "transaction output value of %d is "+
				"higher than max allowed value of %d", sompi, constants.MaxSompi)
		}

		// Binary arithmetic guarantees that any overflow is detected and reported.
		// This is impossible for Hoosat, but perhaps possible if an alt increases
		// the total money supply.
		newTotalSompi := totalSompi + sompi
		if newTotalSompi < totalSompi {
			return errors.Wrapf(ruleerrors.ErrBadTxOutValue, "total value of all transaction "+
				"outputs exceeds max allowed value of %d",
				constants.MaxSompi)
		}
		totalSompi = newTotalSompi
		if totalSompi > constants.MaxSompi {
			return errors.Wrapf(ruleerrors.ErrBadTxOutValue, "total value of all transaction "+
				"outputs is %d which is higher than max "+
				"allowed value of %d", totalSompi,
				constants.MaxSompi)
		}
	}

	return nil
}

func (v *transactionValidator) checkDuplicateTransactionInputs(tx *externalapi.DomainTransaction) error {
	existingTxOut := make(map[externalapi.DomainOutpoint]struct{})
	for _, txIn := range tx.Inputs {
		if _, exists := existingTxOut[txIn.PreviousOutpoint]; exists {
			return errors.Wrapf(ruleerrors.ErrDuplicateTxInputs, "transaction "+
				"contains duplicate inputs")
		}
		existingTxOut[txIn.PreviousOutpoint] = struct{}{}
	}
	return nil
}

func (v *transactionValidator) checkCoinbaseInIsolation(tx *externalapi.DomainTransaction) error {
	if !transactionhelper.IsCoinBase(tx) {
		return nil
	}

	// Coinbase payload length must not exceed the max length.
	payloadLen := len(tx.Payload)
	if uint64(payloadLen) > v.maxCoinbasePayloadLength {
		return errors.Wrapf(ruleerrors.ErrBadCoinbasePayloadLen, "coinbase transaction payload length "+
			"of %d is out of range (max: %d)",
			payloadLen, v.maxCoinbasePayloadLength)
	}

	if len(tx.Inputs) != 0 {
		return errors.Wrap(ruleerrors.ErrCoinbaseWithInputs, "coinbase has inputs")
	}

	outputsLimit := uint64(v.ghostdagK[constants.BlockVersion-1]) + 2
	// Make the outputsLimits twice the size because of developer fee in outputs with coinbase outputs.
	outputsLimit *= 2
	if uint64(len(tx.Outputs)) > outputsLimit {
		return errors.Wrapf(ruleerrors.ErrCoinbaseTooManyOutputs, "coinbase has too many outputs: got %d where the limit is %d", len(tx.Outputs), outputsLimit)
	}

	for i, output := range tx.Outputs {
		if len(output.ScriptPublicKey.Script) > int(v.coinbasePayloadScriptPublicKeyMaxLength) {
			return errors.Wrapf(ruleerrors.ErrCoinbaseTooLongScriptPublicKey, "coinbase output %d has a too long script public key", i)

		}
	}

	return nil
}

func (v *transactionValidator) checkGasInBuiltInOrNativeTransactions(tx *externalapi.DomainTransaction) error {
	// Transactions in native, registry and coinbase subnetworks must have Gas = 0
	if subnetworks.IsBuiltInOrNative(tx.SubnetworkID) && tx.Gas > 0 {
		return errors.Wrapf(ruleerrors.ErrInvalidGas, "transaction in the native or "+
			"registry subnetworks has gas > 0 ")
	}
	return nil
}

func (v *transactionValidator) checkSubnetworkRegistryTransaction(tx *externalapi.DomainTransaction) error {
	if tx.SubnetworkID != subnetworks.SubnetworkIDRegistry {
		return nil
	}

	if len(tx.Payload) != 8 {
		return errors.Wrapf(ruleerrors.ErrSubnetworkRegistry, "validation failed: subnetwork registry "+
			"tx has an invalid payload")
	}
	return nil
}

func (v *transactionValidator) checkNativeTransactionPayload(tx *externalapi.DomainTransaction) error {
	if tx.SubnetworkID == subnetworks.SubnetworkIDNative && len(tx.Payload) > 0 {
		return errors.Wrapf(ruleerrors.ErrInvalidPayload, "transaction in the native subnetwork "+
			"includes a payload")
	}
	return nil
}

func IsValidJSONObject(data []byte) (bool, error) {
	if len(data) == 0 {
		return false, fmt.Errorf("empty input data")
	}

	var obj map[string]interface{}
	err := json.Unmarshal(data, &obj)
	if err != nil {
		return false, fmt.Errorf("invalid JSON: %v", err)
	}

	if obj == nil {
		return false, fmt.Errorf("input is not a JSON object")
	}

	if containsBinaryOrImage(obj) {
		return false, fmt.Errorf("contains binary or image data")
	}

	return true, nil
}

func containsBinaryOrImage(data interface{}) bool {
	switch v := data.(type) {
	case map[string]interface{}:
		for _, val := range v {
			if containsBinaryOrImage(val) {
				return true
			}
		}
	case []interface{}:
		for _, val := range v {
			if containsBinaryOrImage(val) {
				return true
			}
		}
	case string:
		if decoded, err := base64.StdEncoding.DecodeString(v); err == nil {
			if isImage(decoded) || len(decoded) > 0 {
				return true
			}
		}
		if decoded, err := base64.URLEncoding.DecodeString(v); err == nil {
			if isImage(decoded) || len(decoded) > 0 {
				return true
			}
		}

		if isHexString(v) {
			if decoded, err := hex.DecodeString(strings.ToLower(v)); err == nil {
				if isImage(decoded) || len(decoded) > 0 {
					return true
				}
			}
		}

		if isAscii85String(v) {
			decoded := make([]byte, len(v)*4/5+4)
			n, _, err := ascii85.Decode(decoded, []byte(v), true)
			if err == nil {
				decoded = decoded[:n]
				if isImage(decoded) || len(decoded) > 0 {
					return true
				}
			}
		}
	case []byte: // Direct byte slices are considered binary
		return true
	}
	return false
}

func isImage(data []byte) bool {
	_, _, err := image.Decode(bytes.NewReader(data))
	return err == nil
}

func isHexString(s string) bool {
	if len(s)%2 != 0 {
		return false
	}
	matched, _ := regexp.MatchString(`^[0-9a-fA-F]+$`, s)
	return matched
}

func isAscii85String(s string) bool {
	if len(s) < 5 {
		return false
	}
	for _, r := range s {
		if r < '!' || r > 'u' {
			return false
		}
	}
	return true
}

func (v *transactionValidator) checkDataTransactionPayload(tx *externalapi.DomainTransaction) error {
	if tx.SubnetworkID != subnetworks.SubnetworkIDData || len(tx.Payload) <= 0 {
		return nil
	}

	payloadString := strings.TrimSpace(string(tx.Payload))
	if len(payloadString)%2 != 0 {
		fmt.Fprintf(os.Stderr, "Error: Invalid hex string length\n")
		return errors.Wrapf(ruleerrors.ErrInvalidPayload, "transaction in the native subnetwork "+
			"includes a payload")
	}

	payload, err := hex.DecodeString(payloadString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding hex: %v\n", err)
		os.Exit(1)
	}

	if isValid, err := IsValidJSONObject(payload); !isValid {
		return errors.Wrapf(ruleerrors.ErrInvalidPayload, "data subnetwork transaction payload is not valid JSON: %v", err)
	}

	return nil
}

func (v *transactionValidator) checkTransactionSubnetwork(tx *externalapi.DomainTransaction,
	localNodeSubnetworkID *externalapi.DomainSubnetworkID) error {
	if !v.enableNonNativeSubnetworks &&
		tx.SubnetworkID != subnetworks.SubnetworkIDNative &&
		tx.SubnetworkID != subnetworks.SubnetworkIDCoinbase &&
		tx.SubnetworkID != subnetworks.SubnetworkIDData {
		return errors.Wrapf(ruleerrors.ErrSubnetworksDisabled, "transaction has non native or coinbase "+
			"subnetwork ID")
	}

	// If we are a partial node, only transactions on built in subnetworks
	// or our own subnetwork may have a payload
	isLocalNodeFull := localNodeSubnetworkID == nil
	shouldTxBeFull := subnetworks.IsBuiltIn(tx.SubnetworkID) || tx.SubnetworkID.Equal(localNodeSubnetworkID)
	if !isLocalNodeFull && !shouldTxBeFull && len(tx.Payload) > 0 {
		return errors.Wrapf(ruleerrors.ErrInvalidPayload,
			"transaction that was expected to be partial has a payload "+
				"with length > 0")
	}
	return nil
}
