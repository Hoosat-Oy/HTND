package pow

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/consensushashing"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/hashes"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/serialization"
	"github.com/Hoosat-Oy/HTND/util/difficulty"
	"golang.org/x/crypto/blake2b"

	"math/big"

	"github.com/pkg/errors"
)

const tableSize = 1 << 20 // 64 KB table (reduced from 16 MB)
var lookupTable [tableSize]uint64

func generateHoohashLookupTable() {
	// Initialize lookup table deterministically
	var seed [32]byte
	for i := range lookupTable {
		// Use SHA-256 to generate deterministic values
		binary.BigEndian.PutUint32(seed[:], uint32(i))
		hash := sha256.Sum256(seed[:])
		lookupTable[i] = binary.BigEndian.Uint64(hash[:8])
	}
}

func timeMemoryTradeoff(input uint64) uint64 {
	result := input
	for i := 0; i < 1000; i++ { // Number of lookups
		index := result % tableSize
		result ^= lookupTable[index]
		result = (result << 1) | (result >> 63) // Rotate left by 1
	}
	return result
}

// State is an intermediate data structure with pre-computed values to speed up mining.
type State struct {
	mat          matrix
	Timestamp    int64
	Nonce        uint64
	Target       big.Int
	prePowHash   externalapi.DomainHash
	blockVersion uint16
}

// NewState creates a new state with pre-computed values to speed up mining
// It takes the target from the Bits field
func NewState(header externalapi.MutableBlockHeader) *State {
	target := difficulty.CompactToBig(header.Bits())
	// Zero out the time and nonce.
	timestamp, nonce := header.TimeInMilliseconds(), header.Nonce()
	header.SetTimeInMilliseconds(0)
	header.SetNonce(0)
	prePowHash := consensushashing.HeaderHash(header)
	header.SetTimeInMilliseconds(timestamp)
	header.SetNonce(nonce)
	if header.Version() == 2 {
		return &State{
			Target:       *target,
			prePowHash:   *prePowHash,
			mat:          *GenerateHoohashMatrix(prePowHash),
			Timestamp:    timestamp,
			Nonce:        nonce,
			blockVersion: header.Version(),
		}
	}
	return &State{
		Target:       *target,
		prePowHash:   *prePowHash,
		mat:          *GenerateMatrix(prePowHash),
		Timestamp:    timestamp,
		Nonce:        nonce,
		blockVersion: header.Version(),
	}
}

func memoryHardFunction(input []byte) []byte {
	const memorySize = 1 << 10 // 2^16 = 65536
	const iterations = 2

	memory := make([]uint64, memorySize)

	// Initialize memory
	for i := range memory {
		memory[i] = binary.LittleEndian.Uint64(input)
	}

	// Perform memory-hard computations
	for i := 0; i < iterations; i++ {
		for j := 0; j < memorySize; j++ {
			index1 := memory[j] % uint64(memorySize)
			index2 := (memory[j] >> 32) % uint64(memorySize)

			hash, _ := blake2b.New512(nil)
			binary.Write(hash, binary.LittleEndian, memory[index1])
			binary.Write(hash, binary.LittleEndian, memory[index2])

			memory[j] = binary.LittleEndian.Uint64(hash.Sum(nil))
		}
	}

	// Combine results
	result := make([]byte, 64)
	for i := 0; i < 8; i++ {
		binary.LittleEndian.PutUint64(result[i*8:], memory[i])
	}
	return result
}

func verifiableDelayFunction(input []byte) []byte {
	const iterations = 10000000 // Adjust based on desired delay
	// 1000 = 350µs
	// Create a prime field
	p, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F", 16)
	// Convert input to big.Int
	x := new(big.Int).SetBytes(input)
	// Perform repeated squaring
	for i := 0; i < iterations; i++ {
		x.Mul(x, x)
		x.Mod(x, p)
	}
	hash := sha256.Sum256(x.Bytes())
	return hash[:]
}
func (state *State) CalculateProofOfWorkValue() (*big.Int, *externalapi.DomainHash) {
	if state.blockVersion == 1 {
		return state.CalculateProofOfWorkValuePyrinhash()
	} else if state.blockVersion == 2 {
		return state.CalculateProofOfWorkValueHoohashV1()
	} else if state.blockVersion == 3 {
		return state.CalculateProofOfWorkValueHoohashV101()
	} else {
		return state.CalculateProofOfWorkValuePyrinhash() // default to the oldest version.
	}
}

func (state *State) CalculateProofOfWorkValueHoohashV2() (*big.Int, *externalapi.DomainHash) {
	// PRE_POW_HASH || TIME || 32 zero byte padding || NONCE
	writer := hashes.Blake3HashWriter()
	writer.InfallibleWrite(state.prePowHash.ByteSlice())
	err := serialization.WriteElement(writer, state.Timestamp)
	if err != nil {
		panic(errors.Wrap(err, "this should never happen. Hash digest should never return an error"))
	}
	zeroes := [32]byte{}
	writer.InfallibleWrite(zeroes[:])
	err = serialization.WriteElement(writer, state.Nonce)
	if err != nil {
		panic(errors.Wrap(err, "this should never happen. Hash digest should never return an error"))
	}
	powHash := writer.Finalize()
	memoryHardResult := memoryHardFunction(powHash.ByteSlice())
	tradeoffResult := timeMemoryTradeoff(binary.BigEndian.Uint64(memoryHardResult))
	vdfResult := verifiableDelayFunction(memoryHardResult)
	combined := append(memoryHardResult, vdfResult...)
	combined = append(combined, byte(tradeoffResult))
	hash := state.mat.HoohashMatrixMultiplicationV101(externalapi.NewDomainHashFromByteArray((*[32]byte)(combined)))
	return toBig(hash), hash
}

func (state *State) CalculateProofOfWorkValueHoohashV1() (*big.Int, *externalapi.DomainHash) {
	// PRE_POW_HASH || TIME || 32 zero byte padding || NONCE
	writer := hashes.Blake3HashWriter()
	writer.InfallibleWrite(state.prePowHash.ByteSlice())
	err := serialization.WriteElement(writer, state.Timestamp)
	if err != nil {
		panic(errors.Wrap(err, "this should never happen. Hash digest should never return an error"))
	}
	zeroes := [32]byte{}
	writer.InfallibleWrite(zeroes[:])
	err = serialization.WriteElement(writer, state.Nonce)
	if err != nil {
		panic(errors.Wrap(err, "this should never happen. Hash digest should never return an error"))
	}
	powHash := writer.Finalize()
	multiplied := state.mat.HoohashMatrixMultiplicationV1(powHash)
	return toBig(multiplied), multiplied
}

func (state *State) CalculateProofOfWorkValueHoohashV101() (*big.Int, *externalapi.DomainHash) {
	// PRE_POW_HASH || TIME || 32 zero byte padding || NONCE
	writer := hashes.Blake3HashWriter()
	writer.InfallibleWrite(state.prePowHash.ByteSlice())
	err := serialization.WriteElement(writer, state.Timestamp)
	if err != nil {
		panic(errors.Wrap(err, "this should never happen. Hash digest should never return an error"))
	}
	zeroes := [32]byte{}
	writer.InfallibleWrite(zeroes[:])
	err = serialization.WriteElement(writer, state.Nonce)
	if err != nil {
		panic(errors.Wrap(err, "this should never happen. Hash digest should never return an error"))
	}
	powHash := writer.Finalize()
	multiplied := state.mat.HoohashMatrixMultiplicationV101(powHash)
	return toBig(multiplied), multiplied
}

// CalculateProofOfWorkValue hashes the internal header and returns its big.Int value
func (state *State) CalculateProofOfWorkValuePyrinhash() (*big.Int, *externalapi.DomainHash) {
	// PRE_POW_HASH || TIME || 32 zero byte padding || NONCE
	writer := hashes.PoWHashWriter() // Blake 3
	writer.InfallibleWrite(state.prePowHash.ByteSlice())
	err := serialization.WriteElement(writer, state.Timestamp)
	if err != nil {
		panic(errors.Wrap(err, "this should never happen. Hash digest should never return an error"))
	}
	zeroes := [32]byte{}
	writer.InfallibleWrite(zeroes[:])
	err = serialization.WriteElement(writer, state.Nonce)
	if err != nil {
		panic(errors.Wrap(err, "this should never happen. Hash digest should never return an error"))
	}
	powHash := writer.Finalize()
	hash := state.mat.bHeavyHash(powHash)
	return toBig(hash), hash
}

// IncrementNonce the nonce in State by 1
func (state *State) IncrementNonce() {
	state.Nonce++
}

// CheckProofOfWork check's if the block has a valid PoW according to the provided target
// it does not check if the difficulty itself is valid or less than the maximum for the appropriate network
func (state *State) CheckProofOfWork(powString string) bool {
	// The block pow must be less than the claimed target
	powNum, _ := state.CalculateProofOfWorkValue()
	if state.blockVersion <= 2 {
		return powNum.Cmp(&state.Target) <= 0
	} else if state.blockVersion >= 3 {
		if powString == "SKIP_POW" {
			return powNum.Cmp(&state.Target) <= 0
		}
		powHash, err := externalapi.NewDomainHashFromString(powString)
		if err != nil {
			return false
		}
		if !powHash.Equal(new(externalapi.DomainHash)) { // Check that PowHash is not empty default.
			submittedPowNum := toBig(powHash)
			if submittedPowNum.Cmp(powNum) == 0 {
				// The block hash must be less or equal than the claimed target, powHash was valid.
				return powNum.Cmp(&state.Target) <= 0
			}
		}
	}
	return false
}

// CheckProofOfWorkByBits check's if the block has a valid PoW according to its Bits field
// it does not check if the difficulty itself is valid or less than the maximum for the appropriate network
func CheckProofOfWorkByBits(header externalapi.MutableBlockHeader, powHash string) bool {
	return NewState(header).CheckProofOfWork(powHash)
}

// ToBig converts a externalapi.DomainHash into a big.Int treated as a little endian string.
func toBig(hash *externalapi.DomainHash) *big.Int {
	// We treat the Hash as little-endian for PoW purposes, but the big package wants the bytes in big-endian, so reverse them.
	buf := hash.ByteSlice()
	blen := len(buf)
	for i := 0; i < blen/2; i++ {
		buf[i], buf[blen-1-i] = buf[blen-1-i], buf[i]
	}

	return new(big.Int).SetBytes(buf)
}

// BlockLevel returns the block level of the given header.
func BlockLevel(header externalapi.BlockHeader, maxBlockLevel int) int {
	// Genesis is defined to be the root of all blocks at all levels, so we define it to be the maximal
	// block level.
	if len(header.DirectParents()) == 0 {
		return maxBlockLevel
	}

	proofOfWorkValue, _ := NewState(header.ToMutable()).CalculateProofOfWorkValue()
	level := maxBlockLevel - proofOfWorkValue.BitLen()
	// If the block has a level lower than genesis make it zero.
	if level < 0 {
		level = 0
	}
	return level
}
