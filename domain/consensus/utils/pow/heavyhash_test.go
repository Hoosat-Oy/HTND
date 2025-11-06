package pow

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"testing"

	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/hashes"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/serialization"
	"golang.org/x/crypto/blake2b"
)

func BenchmarkMatrixHoohashRev2(b *testing.B) {
	input := []byte("BenchmarkMatrix_HeavyHash")
	generateHoohashLookupTable()
	for i := 0; i < b.N; i++ {
		firstPass := hashes.Blake3HashWriter()
		firstPass.InfallibleWrite(input)
		hash := firstPass.Finalize()
		// memoryHardResult := memoryHardFunction(hash.ByteSlice())
		// tradeoffResult := timeMemoryTradeoff(binary.BigEndian.Uint64(memoryHardResult))
		verifiableDelayFunction(hash.ByteSlice())
		// combined := append(memoryHardResult, vdfResult...)

		// matrix := GenerateHoohashMatrix(hash)
		// hash = matrix.HoohashMatrixMultiplicationV1(externalapi.NewDomainHashFromByteArray((*[32]byte)(combined)))
	}
}

func BenchmarkCreatLUT(b *testing.B) {
	forComplexMap := map[int]float64{}
	fmt.Println("forComplexMap := map[float64]float64 {")
	for i := 0; i < 256; i++ {
		forComplex := float64(i)
		for forComplex > 14 {
			forComplex = forComplex * 0.1
		}
		forComplexMap[i] = ComplexNonLinear(forComplex)
		fmt.Printf("\t%d: %f,\n", i, forComplexMap[i])
	}
	fmt.Println("}")
	b.Fail()
}

func GenerateHoohashMatrixTest(hash *externalapi.DomainHash, t *testing.T) *matrix {
	var mat matrix
	generator := newxoShiRo256PlusPlus(hash)
	// fmt.Printf("Generator.s0 %x\n", generator.s0)
	// fmt.Printf("Generator.s1 %x\n", generator.s1)
	// fmt.Printf("Generator.s2 %x\n", generator.s2)
	// fmt.Printf("Generator.s3 %x\n", generator.s3)
	for {
		for i := range mat {
			for j := 0; j < 64; j += 16 {
				val := generator.Uint64()
				mat[i][j] = uint16(val & 0x0F)
				mat[i][j+1] = uint16((val >> 4) & 0x0F)
				mat[i][j+2] = uint16((val >> 8) & 0x0F)
				mat[i][j+3] = uint16((val >> 12) & 0x0F)
				mat[i][j+4] = uint16((val >> 16) & 0x0F)
				mat[i][j+5] = uint16((val >> 20) & 0x0F)
				mat[i][j+6] = uint16((val >> 24) & 0x0F)
				mat[i][j+7] = uint16((val >> 28) & 0x0F)
				mat[i][j+8] = uint16((val >> 32) & 0x0F)
				mat[i][j+9] = uint16((val >> 36) & 0x0F)
				mat[i][j+10] = uint16((val >> 40) & 0x0F)
				mat[i][j+11] = uint16((val >> 44) & 0x0F)
				mat[i][j+12] = uint16((val >> 48) & 0x0F)
				mat[i][j+13] = uint16((val >> 52) & 0x0F)
				mat[i][j+14] = uint16((val >> 56) & 0x0F)
				mat[i][j+15] = uint16((val >> 60) & 0x0F)
			}
		}
		rank := mat.computeHoohashRank()
		if rank == 64 {
			return &mat
		}
	}
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

func TestGenerateMatrix102(t *testing.T) {
	prePowHash, _ := hex.DecodeString("82b1d17c5e2200a0565956b711485a2cba6da909e588261582c2f465ec2e3d3f")
	matrix := GenerateHoohashMatrixV110(externalapi.NewDomainHashFromByteArray((*[32]byte)(prePowHash)))
	fmt.Printf("Matrix: %v\n", matrix)
	t.Fail()
}

func ForComplexTest(forComplex float64) float64 {
	var complex float64
	rounds := 1
	complex = ComplexNonLinear110(forComplex)
	for math.IsNaN(float64(complex)) || math.IsInf(float64(complex), 0) {
		forComplex *= 0.1
		if forComplex <= 0.0000000000001 {
			return 0 * float64(rounds)
		}
		rounds++
		complex = ComplexNonLinear110(forComplex)
	}
	// fmt.Printf("%d\n", rounds)
	return complex * float64(rounds)
}

func (mat *floatMatrix) HoohashMatrixMultiplicationV110Test(hash *externalapi.DomainHash, Nonce uint64, t *testing.T) *externalapi.DomainHash {
	hashBytes := hash.ByteArray()
	H := hash.Uint32Array()
	hashMod := float64(H[0] ^ H[1] ^ H[2] ^ H[3] ^ H[4] ^ H[5] ^ H[6] ^ H[7])
	nonceMod := float64(Nonce & 0xFF)
	divider := 0.0001
	multiplier := float64(1234)
	var vector [64]byte
	var product [64]float64
	sw := float64(0.0)

	// Populate the vector with floating-point values from the hash bytes
	for i := 0; i < 32; i++ {
		vector[2*i] = hashBytes[i] >> 4     // Upper 4 bits
		vector[2*i+1] = hashBytes[i] & 0x0F // Lower 4 bits
	}

	// fmt.Printf("Vector: %v\n\n", vector)

	// Perform the matrix-vector multiplication with nonlinear adjustments
	for i := 0; i < 64; i++ {
		for j := 0; j < 64; j++ {
			if sw <= 0.02 {
				input := (mat[i][j]*hashMod*float64(vector[j]) + nonceMod)
				output := ForComplexTest(input) * float64(vector[j]) * multiplier
				product[i] += output
				//fmt.Printf("[%d][%d]: %f %f %f %f %f %f\n", i, j, mat[i][j], float64(vector[j]), hashMod, nonceMod, input, output)
			} else {
				product[i] += mat[i][j] * divider * float64(vector[j])
			}
			sw = TransformFactor(product[i])
		}
	}
	fmt.Printf("\n")

	// Generate the result bytes

	var res [32]uint8
	var scaledValues [32]uint8
	for i := 0; i < 64; i += 2 {
		pval := uint64(product[i]) + uint64(product[i+1])
		scaledValues[i/2] = uint8(pval & 0xFF)
		// fmt.Printf("[%d] -> %f + %f -> %d -> %d\n", i/2, product[i], product[i+1], pval, scaledValues[i/2])
	}
	// fmt.Printf("Final pass: [")
	for i := 0; i < 32; i++ {
		res[i] = hashBytes[i] ^ scaledValues[i]
		fmt.Printf("%d, ", res[i])
	}
	// fmt.Printf("]\n")
	writer := hashes.Blake3HashWriter()
	writer.InfallibleWrite(res[:32])
	return writer.Finalize()
}

func TestMatrixHoohashRev110(t *testing.T) {
	file, err := os.Create("pow_hashes.txt")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()
	for i := 0; i < 1000; i++ {
		for x := 0; x < 1000; x++ {
			fmt.Printf("------------------------------\n")
			Nonce := uint64(x)
			fmt.Printf("Test %d\n", int64(i))
			prePowHash, _ := hex.DecodeString("b7c8f43d8a99aecdd37912c9ad4f2e51c8009f7ce1cdf6e3be2767972cc68a1c")
			Timestamp := int64(172537456 + i)
			// PRE_POW_HASH || TIME || 32 zero byte padding || NONCE
			writer := hashes.Blake3HashWriter()
			fmt.Printf("PRE_POW_HASH: %v\n", hex.EncodeToString(prePowHash))
			writer.InfallibleWrite(prePowHash)
			fmt.Printf("TIME: %d\n", Timestamp)
			_ = serialization.WriteElement(writer, Timestamp)
			zeroes := [32]byte{}
			writer.InfallibleWrite(zeroes[:])
			fmt.Printf("Nonce: %d\n", Nonce)
			_ = serialization.WriteElement(writer, Nonce)
			powHash := writer.Finalize()
			matrix := GenerateHoohashMatrixV110(externalapi.NewDomainHashFromByteArray((*[32]byte)(prePowHash)))
			//fmt.Printf("Matrix: %v\n", matrix)
			multiplied := matrix.HoohashMatrixMultiplicationV110Test(powHash, Nonce, t)
			fmt.Printf("POW HASH: %v\n", multiplied)
			_, err := file.WriteString(fmt.Sprintf("%v\n", multiplied))
			if err != nil {
				fmt.Println("Error writing to file:", err)
				return
			}
		}
	}
	fmt.Printf("------------------------------\n")
	Nonce := uint64(7794931619413402210)
	prePowHash, _ := hex.DecodeString("82b1d17c5e2200a0565956b711485a2cba6da909e588261582c2f465ec2e3d3f")
	Timestamp := int64(1727011258677)
	// PRE_POW_HASH || TIME || 32 zero byte padding || NONCE
	writer := hashes.Blake3HashWriter()
	fmt.Printf("PRE_POW_HASH: %v\n", hex.EncodeToString(prePowHash))
	writer.InfallibleWrite(prePowHash)
	fmt.Printf("TIME: %d\n", Timestamp)
	_ = serialization.WriteElement(writer, Timestamp)
	zeroes := [32]byte{}
	writer.InfallibleWrite(zeroes[:])
	fmt.Printf("Nonce: %d\n", Nonce)
	_ = serialization.WriteElement(writer, Nonce)
	powHash := writer.Finalize()
	matrix := GenerateHoohashMatrixV110(externalapi.NewDomainHashFromByteArray((*[32]byte)(prePowHash)))
	//fmt.Printf("Matrix: %v\n\n", matrix[0])
	multiplied := matrix.HoohashMatrixMultiplicationV110Test(powHash, Nonce, t)
	fmt.Printf("POW HASH: %v\n", multiplied)
	t.Fail()
}

func (mat *matrix) HoohashMatrixMultiplicationTest(hash *externalapi.DomainHash, t *testing.T) *externalapi.DomainHash {
	hashBytes := hash.ByteArray()
	var vector [64]float64
	var product [64]float64

	// Populate the vector with floating-point values
	for i := 0; i < 32; i++ {
		vector[2*i] = float64(hashBytes[i] >> 4)
		vector[2*i+1] = float64(hashBytes[i] & 0x0F)
	}
	// fmt.Printf("Vector: ")
	// for i := 0; i < 64; i++ {
	// 	fmt.Printf("%f, ", vector[i])
	// }
	// fmt.Printf("\n")

	// fmt.Printf("Matrix first 64: ")
	// for i := 0; i < 64; i++ {
	// 	fmt.Printf("%d, ", mat[0][i])
	// }
	// fmt.Printf("\n")

	// forComplexMap := map[int]float64{
	// 	0:   3.981957,
	// 	1:   0.061209,
	// 	2:   0.019915,
	// 	3:   0.979552,
	// 	4:   0.919536,
	// 	5:   0.326682,
	// 	6:   0.169842,
	// 	7:   0.572750,
	// 	8:   0.169842,
	// 	9:   0.989227,
	// 	10:  9063640.699722,
	// 	11:  1066.133994,
	// 	12:  1167554.304901,
	// 	13:  322655.148236,
	// 	14:  4076515423.444545,
	// 	15:  0.015544,
	// 	16:  0.039470,
	// 	17:  0.073738,
	// 	18:  0.117579,
	// 	19:  0.170008,
	// 	20:  0.019915,
	// 	21:  0.003705,
	// 	22:  0.070329,
	// 	23:  0.213984,
	// 	24:  0.413447,
	// 	25:  0.633856,
	// 	26:  0.831949,
	// 	27:  0.964373,
	// 	28:  0.997735,
	// 	29:  0.918170,
	// 	30:  0.979552,
	// 	31:  0.969452,
	// 	32:  0.957767,
	// 	33:  0.944682,
	// 	34:  0.930368,
	// 	35:  0.914985,
	// 	36:  0.898682,
	// 	37:  0.881594,
	// 	38:  0.863847,
	// 	39:  0.845554,
	// 	40:  0.919536,
	// 	41:  0.839197,
	// 	42:  0.737768,
	// 	43:  0.621557,
	// 	44:  0.497787,
	// 	45:  0.374155,
	// 	46:  0.258348,
	// 	47:  0.157565,
	// 	48:  0.078073,
	// 	49:  0.024815,
	// 	50:  0.326682,
	// 	51:  0.398715,
	// 	52:  0.473022,
	// 	53:  0.547935,
	// 	54:  0.621772,
	// 	55:  0.692874,
	// 	56:  0.759644,
	// 	57:  0.820584,
	// 	58:  0.874323,
	// 	59:  0.919657,
	// 	60:  0.169842,
	// 	61:  0.014878,
	// 	62:  0.033917,
	// 	63:  0.227935,
	// 	64:  0.529806,
	// 	65:  0.823883,
	// 	66:  0.988895,
	// 	67:  0.949383,
	// 	68:  0.713849,
	// 	69:  0.379665,
	// 	70:  0.572750,
	// 	71:  0.572750,
	// 	72:  0.572750,
	// 	73:  0.572750,
	// 	74:  0.572750,
	// 	75:  0.572750,
	// 	76:  0.572750,
	// 	77:  0.572750,
	// 	78:  0.572750,
	// 	79:  0.572750,
	// 	80:  0.169842,
	// 	81:  0.087207,
	// 	82:  0.030238,
	// 	83:  0.002476,
	// 	84:  0.005648,
	// 	85:  0.039556,
	// 	86:  0.102093,
	// 	87:  0.189369,
	// 	88:  0.295959,
	// 	89:  0.415235,
	// 	90:  0.989227,
	// 	91:  0.999160,
	// 	92:  0.997884,
	// 	93:  0.985426,
	// 	94:  0.962066,
	// 	95:  0.928330,
	// 	96:  0.884974,
	// 	97:  0.832972,
	// 	98:  0.773493,
	// 	99:  0.707871,
	// 	100: 9063640.699722,
	// 	101: 13676676.345494,
	// 	102: 20737135.188099,
	// 	103: 31594405.686592,
	// 	104: 48369060.948050,
	// 	105: 74408715.515206,
	// 	106: 115022066.608368,
	// 	107: 178666088.921568,
	// 	108: 278874615.048666,
	// 	109: 437405159.250589,
	// 	110: 1066.133994,
	// 	111: 1030.056271,
	// 	112: 996.137256,
	// 	113: 964.206073,
	// 	114: 934.108346,
	// 	115: 905.704340,
	// 	116: 878.867343,
	// 	117: 853.482243,
	// 	118: 829.444288,
	// 	119: 806.657993,
	// 	120: 1167554.304901,
	// 	121: 1327468.562041,
	// 	122: 1509224.002327,
	// 	123: 1715796.780501,
	// 	124: 1950567.800298,
	// 	125: 2217377.667536,
	// 	126: 2520589.093506,
	// 	127: 2865157.757474,
	// 	128: 3256712.773579,
	// 	129: 3701648.062254,
	// 	130: 322655.148236,
	// 	131: 348586.196256,
	// 	132: 376594.236118,
	// 	133: 406845.171793,
	// 	134: 439518.129005,
	// 	135: 474806.506913,
	// 	136: 512919.113319,
	// 	137: 554081.390007,
	// 	138: 598536.735370,
	// 	139: 646547.932043,
	// 	140: 4076515423.444545,
	// 	141: 0.003303,
	// 	142: 0.004219,
	// 	143: 0.005247,
	// 	144: 0.006386,
	// 	145: 0.007637,
	// 	146: 0.008998,
	// 	147: 0.010470,
	// 	148: 0.012051,
	// 	149: 0.013743,
	// 	150: 0.015544,
	// 	151: 0.017454,
	// 	152: 0.019472,
	// 	153: 0.021599,
	// 	154: 0.023833,
	// 	155: 0.026175,
	// 	156: 0.028623,
	// 	157: 0.031177,
	// 	158: 0.033836,
	// 	159: 0.036601,
	// 	160: 0.039470,
	// 	161: 0.042442,
	// 	162: 0.045517,
	// 	163: 0.048695,
	// 	164: 0.051974,
	// 	165: 0.055354,
	// 	166: 0.058834,
	// 	167: 0.062413,
	// 	168: 0.066090,
	// 	169: 0.069866,
	// 	170: 0.073738,
	// 	171: 0.077706,
	// 	172: 0.081769,
	// 	173: 0.085926,
	// 	174: 0.090176,
	// 	175: 0.094518,
	// 	176: 0.098952,
	// 	177: 0.103476,
	// 	178: 0.108089,
	// 	179: 0.112790,
	// 	180: 0.117579,
	// 	181: 0.122453,
	// 	182: 0.127413,
	// 	183: 0.132456,
	// 	184: 0.137582,
	// 	185: 0.142789,
	// 	186: 0.148077,
	// 	187: 0.153444,
	// 	188: 0.158889,
	// 	189: 0.164411,
	// 	190: 0.170008,
	// 	191: 0.175680,
	// 	192: 0.181424,
	// 	193: 0.187241,
	// 	194: 0.193127,
	// 	195: 0.199083,
	// 	196: 0.205106,
	// 	197: 0.211196,
	// 	198: 0.217350,
	// 	199: 0.223568,
	// 	200: 0.019915,
	// 	201: 0.014706,
	// 	202: 0.010265,
	// 	203: 0.006606,
	// 	204: 0.003740,
	// 	205: 0.001677,
	// 	206: 0.000428,
	// 	207: 0.000000,
	// 	208: 0.000400,
	// 	209: 0.001634,
	// 	210: 0.003705,
	// 	211: 0.006617,
	// 	212: 0.010369,
	// 	213: 0.014963,
	// 	214: 0.020397,
	// 	215: 0.026666,
	// 	216: 0.033767,
	// 	217: 0.041692,
	// 	218: 0.050434,
	// 	219: 0.059984,
	// 	220: 0.070329,
	// 	221: 0.081458,
	// 	222: 0.093356,
	// 	223: 0.106007,
	// 	224: 0.119393,
	// 	225: 0.133497,
	// 	226: 0.148296,
	// 	227: 0.163768,
	// 	228: 0.179891,
	// 	229: 0.196638,
	// 	230: 0.213984,
	// 	231: 0.231899,
	// 	232: 0.250354,
	// 	233: 0.269318,
	// 	234: 0.288759,
	// 	235: 0.308643,
	// 	236: 0.328936,
	// 	237: 0.349600,
	// 	238: 0.370599,
	// 	239: 0.391895,
	// 	240: 0.413447,
	// 	241: 0.435217,
	// 	242: 0.457162,
	// 	243: 0.479240,
	// 	244: 0.501409,
	// 	245: 0.523625,
	// 	246: 0.545845,
	// 	247: 0.568023,
	// 	248: 0.590114,
	// 	249: 0.612074,
	// 	250: 0.633856,
	// 	251: 0.655416,
	// 	252: 0.676707,
	// 	253: 0.697684,
	// 	254: 0.718301,
	// 	255: 0.738513,
	// }

	// Matrix-vector multiplication with floating point operations
	// fmt.Printf("Product: ")
	for i := 0; i < 64; i++ {
		for j := 0; j < 64; j++ {
			// Transform Matrix values with complex non linear equations and sum into product.
			forComplex := float64(mat[i][j]) * vector[j]
			for forComplex > 14 {
				forComplex = forComplex * 0.1
			}
			product[i] += ComplexNonLinear(float64(forComplex))
			// d := float64(mat[i][j]) * vector[j]
			// forComplex := d
			// for forComplex > 14 {
			// 	forComplex = forComplex * 0.1
			// }
			// complexOutput, exists := forComplexMap[int(d)]
			// if exists {
			// 	product[i] += complexOutput
			// } else {
			// 	product[i] += ComplexNonLinear(forComplex)
			// }
		}
		// fmt.Printf("%.20f, ", product[i])
	}
	// fmt.Printf("\n")

	// Convert product back to uint16 and then to byte array
	var res [32]byte
	// fmt.Printf("Hi/low: ")
	for i := range res {
		high := uint32(product[2*i] * 0.00000001)
		low := uint32(product[2*i+1] * 0.00000001)
		// fmt.Printf("%d - %d, ", high, low)
		// Combine high and low into a single byte
		combined := (high ^ low) & 0xFF
		res[i] = hashBytes[i] ^ byte(combined)
	}
	// fmt.Printf("\n")
	// fmt.Printf("Res: ")
	// for i := range res {
	// 	fmt.Printf("%d, ", res[i])
	// }
	// fmt.Printf("\n")
	// Hash again
	writer := hashes.Blake3HashWriter()
	writer.InfallibleWrite(res[:])
	return writer.Finalize()
}

func TestMatrixHoohashRev1(t *testing.T) {
	for i := 0; i < 100000; i++ {
		fmt.Printf("------------------------------\n")
		Nonce := int64(i)
		fmt.Printf("Test %d\n", int64(i))
		prePowHash, _ := hex.DecodeString("82b1d17c5e2200a0565956b711485a2cba6da909e588261582c2f465ec2e3d3f")
		Timestamp := int64(1727011258677)
		// PRE_POW_HASH || TIME || 32 zero byte padding || NONCE
		writer := hashes.Blake3HashWriter()
		// fmt.Printf("PRE_POW_HASH: %v\n", hex.EncodeToString(prePowHash))
		writer.InfallibleWrite(prePowHash)
		// fmt.Printf("TIME: %d\n", Timestamp)
		serialization.WriteElement(writer, Timestamp)
		zeroes := [32]byte{}
		writer.InfallibleWrite(zeroes[:])
		// fmt.Printf("Nonce: %d\n", Nonce)
		serialization.WriteElement(writer, Nonce)
		powHash := writer.Finalize()
		// fmt.Printf("Powhash finalized: %v\n", powHash)
		matrix := GenerateHoohashMatrixTest(externalapi.NewDomainHashFromByteArray((*[32]byte)(prePowHash)), t)
		multiplied := matrix.HoohashMatrixMultiplicationTest(powHash, t)
		fmt.Printf("POW HASH: %v\n", multiplied)
	}

	t.Fail()
}

func (mat *matrix) kHeavyHashTest(hash *externalapi.DomainHash, t *testing.T) *externalapi.DomainHash {
	hashBytes := hash.ByteArray()
	var vector [64]uint16
	var product [64]uint16
	for i := 0; i < 32; i++ {
		vector[2*i] = uint16(hashBytes[i] >> 4)
		vector[2*i+1] = uint16(hashBytes[i] & 0x0F)
	}
	// Matrix-vector multiplication, and convert to 4 bits.
	for i := 0; i < 64; i++ {
		product[i] = 3
	}
	// Concatenate 4 LSBs back to 8 bit xor with sum1
	var res [32]byte
	for i := range res {
		res[i] = hashBytes[i] ^ (byte(product[2*i]<<4) | byte(product[2*i+1]))
	}
	// Hash again
	writer := hashes.KeccakHeavyHashWriter()
	writer.InfallibleWrite(res[:])
	return writer.Finalize()
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
			_ = binary.Write(hash, binary.LittleEndian, memory[index1])
			_ = binary.Write(hash, binary.LittleEndian, memory[index2])

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

func TestKHeavyHash(t *testing.T) {
	fmt.Printf("------------------------------\n")
	for Nonce := int64(0); Nonce <= 10000; Nonce++ {
		fmt.Printf("Test %d\n", Nonce)
		prePowHash, _ := hex.DecodeString("82b1d17c5e2200a0565956b711485a2cba6da909e588261582c2f465ec2e3d3f")
		Timestamp := int64(1727011258677)
		// PRE_POW_HASH || TIME || 32 zero byte padding || NONCE
		writer := hashes.Blake3HashWriter()
		writer.InfallibleWrite(prePowHash)
		serialization.WriteElement(writer, Timestamp)
		zeroes := [32]byte{}
		writer.InfallibleWrite(zeroes[:])
		fmt.Printf("Nonce: %d\n", Nonce)
		serialization.WriteElement(writer, Nonce)
		powHash := writer.Finalize()
		// fmt.Printf("Powhash finalized: %v\n", powHash)
		matrix := GenerateHoohashMatrix(externalapi.NewDomainHashFromByteArray((*[32]byte)(prePowHash)))
		multiplied := matrix.kHeavyHashTest(powHash, t)
		fmt.Printf("POW HASH: %v\n", multiplied)
	}
	t.Fail()
}

func BenchmarkMatrixKheavyHash(b *testing.B) {
	input := []byte("BenchmarkMatrix_HeavyHash")
	for i := 0; i < b.N; i++ {
		writer := hashes.KeccakHeavyHashWriter()
		writer.InfallibleWrite(input)
		hash := writer.Finalize()
		matrix := GenerateMatrix(hash)
		hash = matrix.kHeavyHash(hash)
	}
}

func BenchmarkMatrixKarlsenHash(b *testing.B) {
	input := []byte("BenchmarkMatrix_HeavyHash")
	for i := 0; i < b.N; i++ {
		writer := hashes.BlakeHeavyHashWriter()
		writer.InfallibleWrite(input)
		hash := writer.Finalize()
		matrix := GenerateMatrix(hash)
		hash = matrix.kHeavyHash(hash)
	}
}

func BenchmarkMatrixPyrinhash(b *testing.B) {
	input := []byte("BenchmarkMatrix_HeavyHash")
	for i := 0; i < b.N; i++ {
		writer := hashes.BlakeHeavyHashWriter()
		writer.InfallibleWrite(input)
		hash := writer.Finalize()
		matrix := GenerateMatrix(hash)
		hash = matrix.bHeavyHash(hash)
	}
}

func BenchmarkMatrix_Generate(b *testing.B) {
	r := rand.New(rand.NewSource(0))
	h := [32]byte{}
	r.Read(h[:])
	hash := externalapi.NewDomainHashFromByteArray(&h)
	for i := 0; i < b.N; i++ {
		GenerateMatrix(hash)
	}
}

func BenchmarkMatrix_Rank(b *testing.B) {
	r := rand.New(rand.NewSource(0))
	h := [32]byte{}
	r.Read(h[:])
	hash := externalapi.NewDomainHashFromByteArray(&h)
	matrix := GenerateMatrix(hash)

	for i := 0; i < b.N; i++ {
		matrix.computeRank()
	}
}

func TestMatrix_Rank(t *testing.T) {
	var mat matrix
	if mat.computeRank() != 0 {
		t.Fatalf("The zero matrix should have rank 0, instead got: %d", mat.computeRank())
	}

	r := rand.New(rand.NewSource(0))
	h := [32]byte{}
	r.Read(h[:])
	hash := externalapi.NewDomainHashFromByteArray(&h)
	mat = *GenerateMatrix(hash)

	if mat.computeRank() != 64 {
		t.Fatalf("GenerateMatrix() should always return full rank matrix, instead got: %d", mat.computeRank())
	}

	for i := range mat {
		mat[0][i] = mat[1][i] * 2
	}
	if mat.computeRank() != 63 {
		t.Fatalf("Made a linear depenency between 2 rows, rank should be 63, instead got: %d", mat.computeRank())
	}

	for i := range mat {
		mat[33][i] = mat[32][i] * 3
	}
	if mat.computeRank() != 62 {
		t.Fatalf("Made anoter linear depenency between 2 rows, rank should be 62, instead got: %d", mat.computeRank())
	}
}

func TestGenerateMatrix(t *testing.T) {
	hashBytes := [32]byte{42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42}
	hash := externalapi.NewDomainHashFromByteArray(&hashBytes)
	expectedMatrix := matrix{
		{4, 5, 4, 5, 4, 5, 4, 5, 4, 5, 4, 5, 4, 5, 4, 5, 15, 3, 15, 3, 15, 3, 15, 3, 15, 3, 15, 3, 15, 3, 15, 3, 2, 10, 2, 10, 2, 10, 2, 10, 2, 10, 2, 10, 2, 10, 2, 10, 14, 1, 2, 2, 14, 10, 4, 12, 4, 12, 10, 10, 10, 10, 10, 10},
		{9, 11, 1, 11, 1, 11, 9, 11, 9, 11, 9, 3, 12, 13, 11, 5, 15, 15, 5, 0, 6, 8, 1, 8, 6, 11, 15, 5, 3, 6, 7, 3, 2, 15, 14, 3, 7, 11, 14, 7, 3, 6, 14, 12, 3, 9, 5, 1, 1, 0, 8, 4, 10, 15, 9, 10, 6, 13, 1, 1, 7, 4, 4, 6},
		{2, 6, 0, 8, 11, 15, 4, 0, 5, 2, 7, 13, 15, 3, 11, 12, 6, 2, 1, 8, 13, 4, 11, 4, 10, 14, 13, 2, 6, 15, 10, 6, 6, 5, 6, 9, 3, 3, 3, 1, 9, 12, 12, 15, 6, 0, 1, 5, 7, 13, 14, 1, 10, 10, 5, 14, 4, 0, 12, 13, 2, 15, 8, 4},
		{8, 6, 5, 1, 0, 6, 4, 8, 13, 0, 8, 12, 7, 2, 4, 3, 10, 5, 9, 3, 12, 13, 2, 4, 13, 14, 7, 7, 9, 12, 10, 8, 11, 6, 14, 3, 12, 8, 8, 0, 2, 10, 0, 9, 1, 9, 7, 8, 5, 2, 9, 13, 15, 6, 13, 10, 1, 9, 1, 10, 6, 2, 10, 9},
		{4, 2, 6, 14, 4, 2, 5, 7, 15, 6, 0, 4, 11, 9, 12, 0, 3, 2, 0, 4, 10, 5, 12, 3, 3, 4, 10, 1, 0, 13, 3, 12, 15, 0, 7, 10, 2, 2, 15, 0, 2, 15, 8, 2, 15, 12, 10, 6, 6, 2, 13, 3, 8, 14, 3, 13, 10, 5, 4, 5, 1, 6, 5, 10},
		{0, 3, 13, 12, 11, 4, 11, 13, 1, 12, 4, 11, 15, 14, 13, 4, 7, 1, 3, 0, 10, 3, 8, 8, 1, 2, 5, 14, 4, 5, 14, 1, 1, 3, 3, 1, 5, 15, 7, 5, 11, 8, 8, 12, 10, 5, 7, 9, 2, 10, 13, 11, 4, 2, 12, 15, 10, 6, 6, 0, 6, 6, 3, 12},
		{9, 12, 3, 3, 5, 8, 12, 13, 7, 4, 5, 11, 4, 0, 7, 2, 2, 15, 12, 14, 12, 5, 4, 2, 8, 8, 8, 13, 6, 1, 1, 5, 0, 15, 12, 13, 8, 5, 0, 4, 13, 1, 6, 1, 12, 14, 1, 0, 13, 12, 10, 10, 1, 4, 13, 13, 8, 4, 15, 13, 6, 6, 14, 10},
		{14, 15, 8, 0, 7, 2, 5, 10, 5, 3, 12, 0, 11, 3, 4, 2, 8, 11, 6, 14, 14, 3, 3, 12, 3, 7, 6, 2, 6, 12, 15, 1, 1, 13, 0, 6, 9, 9, 7, 7, 13, 4, 4, 2, 15, 5, 2, 15, 13, 13, 10, 6, 9, 15, 2, 9, 6, 10, 6, 14, 14, 3, 5, 11},
		{6, 4, 7, 8, 11, 0, 13, 11, 0, 7, 0, 0, 13, 6, 3, 11, 15, 14, 10, 2, 7, 8, 13, 14, 8, 15, 10, 8, 14, 6, 10, 14, 3, 11, 5, 11, 13, 5, 3, 12, 3, 0, 2, 0, 6, 14, 4, 12, 4, 4, 8, 15, 7, 8, 12, 11, 3, 9, 5, 13, 10, 14, 13, 4},
		{10, 0, 0, 15, 1, 4, 13, 3, 15, 10, 2, 5, 11, 2, 9, 14, 7, 3, 2, 8, 6, 15, 0, 12, 1, 4, 1, 9, 3, 0, 15, 8, 9, 13, 0, 7, 9, 10, 6, 14, 3, 7, 9, 7, 4, 0, 11, 8, 4, 6, 5, 8, 8, 0, 5, 14, 7, 12, 12, 2, 5, 6, 5, 6},
		{12, 0, 0, 14, 8, 3, 0, 3, 13, 10, 5, 13, 5, 7, 2, 4, 13, 11, 3, 1, 11, 2, 14, 5, 10, 5, 5, 9, 12, 15, 12, 8, 1, 0, 11, 13, 8, 1, 1, 11, 10, 0, 11, 15, 13, 9, 12, 14, 5, 4, 5, 14, 2, 7, 2, 1, 4, 12, 11, 11, 9, 12, 11, 15},
		{3, 15, 9, 8, 13, 12, 15, 7, 8, 7, 14, 6, 10, 3, 0, 5, 2, 2, 6, 6, 3, 2, 5, 12, 11, 2, 10, 11, 13, 3, 9, 7, 7, 6, 8, 15, 14, 14, 11, 11, 9, 7, 1, 3, 8, 5, 11, 11, 1, 2, 15, 8, 13, 8, 11, 4, 1, 5, 3, 12, 5, 3, 7, 7},
		{13, 13, 2, 14, 4, 3, 15, 2, 0, 15, 1, 5, 4, 1, 5, 1, 4, 14, 5, 1, 11, 13, 15, 1, 3, 3, 5, 13, 14, 1, 0, 4, 6, 1, 15, 7, 7, 0, 15, 8, 15, 3, 14, 7, 7, 8, 12, 10, 2, 14, 9, 2, 11, 11, 7, 10, 4, 3, 12, 13, 4, 13, 0, 14},
		{12, 14, 15, 15, 2, 0, 0, 13, 4, 6, 4, 2, 14, 11, 5, 6, 14, 8, 14, 7, 13, 15, 6, 15, 7, 9, 1, 0, 11, 9, 9, 0, 2, 12, 8, 8, 14, 11, 7, 5, 3, 0, 11, 12, 9, 2, 8, 9, 0, 0, 9, 8, 9, 8, 2, 14, 12, 2, 0, 14, 13, 8, 4, 10},
		{7, 10, 1, 15, 12, 14, 7, 4, 7, 13, 4, 8, 13, 12, 1, 7, 10, 6, 5, 14, 14, 3, 14, 4, 11, 14, 6, 12, 15, 12, 15, 12, 4, 5, 9, 8, 7, 7, 3, 0, 5, 7, 3, 8, 4, 4, 7, 5, 6, 12, 13, 0, 12, 10, 2, 5, 14, 9, 6, 4, 13, 13, 14, 5},
		{14, 5, 8, 3, 4, 15, 13, 14, 14, 10, 7, 14, 15, 2, 11, 14, 13, 13, 12, 10, 6, 9, 5, 5, 6, 13, 15, 13, 7, 0, 15, 11, 4, 12, 15, 7, 7, 4, 3, 11, 8, 14, 5, 10, 2, 4, 4, 12, 3, 6, 1, 9, 15, 1, 1, 13, 7, 5, 0, 14, 15, 7, 8, 6},
		{1, 2, 10, 5, 2, 13, 1, 11, 15, 10, 4, 9, 9, 12, 14, 13, 3, 5, 0, 3, 7, 11, 10, 3, 12, 5, 10, 2, 13, 7, 1, 7, 13, 8, 2, 8, 3, 14, 10, 3, 5, 12, 0, 9, 3, 9, 11, 2, 10, 9, 0, 6, 4, 0, 1, 14, 11, 0, 8, 6, 1, 15, 3, 10},
		{13, 9, 0, 5, 8, 7, 12, 15, 10, 10, 5, 1, 1, 7, 6, 1, 14, 5, 15, 2, 3, 5, 3, 5, 7, 3, 7, 7, 1, 4, 3, 14, 5, 0, 12, 0, 12, 10, 10, 6, 12, 6, 3, 5, 5, 11, 10, 1, 11, 3, 13, 3, 9, 11, 1, 7, 14, 14, 0, 8, 15, 5, 2, 7},
		{8, 5, 11, 6, 15, 0, 1, 13, 1, 6, 7, 15, 4, 3, 14, 12, 9, 3, 11, 6, 4, 12, 1, 11, 6, 12, 5, 11, 1, 12, 2, 3, 1, 2, 11, 12, 0, 5, 11, 5, 3, 13, 11, 3, 11, 14, 10, 8, 3, 9, 4, 8, 13, 11, 9, 11, 2, 4, 12, 3, 0, 14, 7, 11},
		{10, 11, 4, 10, 7, 8, 3, 14, 15, 8, 15, 6, 9, 8, 5, 6, 12, 1, 15, 6, 5, 5, 14, 13, 2, 12, 14, 6, 5, 5, 14, 9, 1, 10, 11, 14, 8, 6, 14, 11, 1, 15, 6, 11, 11, 8, 1, 2, 8, 5, 4, 15, 6, 8, 0, 8, 0, 11, 0, 1, 0, 7, 8, 15},
		{0, 15, 5, 0, 11, 4, 4, 2, 0, 4, 8, 12, 2, 2, 0, 8, 1, 2, 6, 5, 6, 12, 3, 1, 12, 1, 6, 10, 2, 5, 0, 2, 0, 11, 8, 6, 13, 4, 14, 4, 15, 5, 8, 11, 9, 6, 2, 6, 9, 1, 4, 2, 14, 10, 4, 4, 1, 1, 11, 8, 6, 11, 11, 9},
		{7, 3, 6, 5, 9, 1, 11, 0, 15, 13, 13, 13, 4, 14, 14, 12, 3, 7, 9, 3, 1, 6, 5, 9, 7, 6, 2, 11, 10, 4, 11, 14, 10, 13, 11, 8, 11, 8, 1, 15, 5, 0, 10, 5, 6, 0, 5, 15, 11, 6, 6, 4, 10, 11, 8, 12, 0, 10, 11, 11, 11, 1, 13, 6},
		{7, 15, 0, 0, 11, 5, 7, 13, 3, 7, 3, 2, 5, 12, 6, 11, 14, 4, 9, 8, 9, 9, 13, 0, 15, 2, 13, 2, 15, 6, 15, 1, 1, 7, 4, 0, 10, 1, 8, 14, 0, 10, 12, 4, 5, 13, 9, 0, 7, 12, 13, 11, 11, 8, 8, 15, 2, 15, 4, 4, 9, 3, 10, 7},
		{0, 9, 3, 5, 14, 6, 7, 14, 7, 2, 13, 7, 3, 15, 9, 15, 2, 8, 0, 4, 6, 0, 15, 6, 2, 1, 14, 8, 5, 8, 2, 4, 2, 11, 9, 2, 15, 13, 11, 12, 8, 15, 3, 13, 2, 2, 10, 13, 1, 8, 7, 15, 13, 6, 7, 7, 4, 3, 14, 7, 0, 9, 15, 11},
		{8, 13, 7, 7, 8, 8, 7, 8, 1, 4, 10, 1, 12, 4, 14, 11, 7, 12, 15, 0, 10, 15, 9, 2, 14, 2, 14, 2, 4, 5, 13, 3, 2, 10, 0, 15, 7, 6, 8, 11, 7, 6, 10, 10, 4, 7, 10, 6, 6, 14, 10, 4, 14, 6, 12, 2, 8, 1, 9, 13, 3, 4, 3, 14},
		{10, 10, 6, 3, 8, 5, 10, 7, 11, 10, 9, 4, 8, 14, 9, 10, 0, 9, 8, 14, 11, 15, 8, 13, 13, 7, 13, 13, 13, 9, 12, 11, 6, 3, 9, 6, 0, 0, 6, 6, 11, 6, 4, 8, 1, 5, 1, 7, 9, 6, 13, 4, 3, 8, 8, 11, 9, 10, 6, 11, 12, 13, 14, 14},
		{14, 10, 0, 15, 14, 4, 3, 0, 12, 4, 0, 14, 11, 9, 0, 6, 4, 6, 0, 9, 8, 14, 4, 4, 6, 8, 2, 8, 10, 3, 8, 0, 1, 1, 15, 4, 2, 4, 13, 9, 9, 4, 0, 5, 5, 1, 2, 5, 11, 6, 2, 1, 7, 8, 10, 10, 1, 5, 8, 6, 7, 0, 4, 14},
		{0, 15, 10, 11, 13, 12, 7, 7, 4, 0, 9, 5, 2, 8, 0, 10, 6, 6, 7, 5, 6, 7, 9, 0, 1, 4, 8, 14, 10, 3, 5, 5, 11, 5, 1, 10, 6, 10, 0, 14, 1, 15, 11, 12, 8, 2, 7, 8, 4, 0, 3, 11, 9, 15, 3, 5, 15, 15, 14, 15, 3, 4, 5, 14},
		{5, 12, 12, 8, 0, 0, 14, 1, 4, 15, 3, 2, 2, 6, 1, 10, 7, 10, 14, 5, 14, 0, 8, 5, 9, 0, 12, 8, 9, 10, 3, 12, 3, 2, 0, 0, 12, 12, 7, 13, 2, 6, 4, 7, 10, 10, 14, 1, 11, 6, 10, 3, 12, 2, 1, 10, 7, 13, 10, 12, 14, 11, 14, 8},
		{9, 5, 3, 12, 4, 3, 10, 14, 7, 5, 11, 12, 2, 13, 9, 8, 5, 2, 6, 2, 4, 9, 10, 10, 4, 3, 4, 0, 11, 1, 10, 9, 4, 10, 4, 5, 8, 11, 1, 7, 13, 7, 6, 6, 3, 12, 0, 0, 15, 6, 12, 12, 13, 7, 14, 14, 11, 15, 7, 14, 12, 6, 15, 2},
		{15, 2, 0, 12, 15, 14, 8, 14, 7, 14, 0, 3, 3, 11, 12, 2, 3, 14, 13, 5, 12, 9, 6, 11, 7, 4, 5, 1, 7, 12, 0, 11, 1, 5, 6, 6, 8, 6, 12, 2, 12, 3, 10, 3, 4, 10, 3, 3, 3, 10, 10, 14, 3, 13, 15, 0, 7, 6, 15, 6, 13, 7, 4, 11},
		{11, 15, 5, 14, 0, 1, 1, 14, 2, 3, 15, 14, 4, 3, 11, 1, 6, 6, 0, 12, 3, 5, 15, 6, 3, 11, 13, 11, 7, 7, 8, 11, 5, 9, 10, 10, 9, 14, 7, 1, 7, 2, 8, 6, 6, 5, 1, 9, 6, 5, 8, 14, 2, 14, 2, 9, 3, 3, 4, 15, 13, 5, 2, 7},
		{7, 8, 13, 9, 15, 8, 11, 7, 1, 9, 15, 12, 6, 9, 3, 1, 10, 10, 11, 0, 0, 8, 14, 5, 11, 12, 14, 4, 3, 9, 12, 9, 14, 0, 0, 9, 12, 4, 1, 13, 3, 6, 3, 4, 13, 10, 2, 9, 3, 7, 7, 10, 7, 10, 10, 3, 5, 15, 8, 9, 11, 7, 1, 14},
		{5, 5, 9, 1, 15, 3, 3, 11, 6, 11, 13, 13, 4, 12, 7, 12, 4, 8, 14, 13, 7, 12, 13, 8, 10, 2, 1, 12, 11, 7, 0, 8, 10, 9, 15, 1, 3, 9, 10, 0, 9, 1, 14, 1, 1, 9, 2, 2, 8, 9, 5, 6, 3, 2, 15, 9, 15, 6, 3, 11, 14, 4, 0, 4},
		{9, 2, 10, 2, 0, 9, 6, 13, 13, 0, 13, 14, 3, 12, 1, 15, 9, 3, 12, 2, 5, 15, 6, 6, 15, 11, 7, 11, 0, 4, 0, 11, 10, 12, 7, 9, 3, 0, 2, 2, 13, 13, 9, 6, 9, 2, 6, 4, 3, 6, 5, 10, 10, 9, 7, 2, 4, 9, 13, 11, 2, 13, 6, 8},
		{13, 15, 9, 8, 6, 2, 3, 2, 2, 12, 5, 3, 8, 6, 11, 6, 15, 7, 10, 3, 15, 8, 7, 5, 3, 8, 4, 2, 11, 1, 0, 4, 1, 1, 6, 1, 13, 6, 5, 1, 2, 6, 7, 10, 4, 3, 10, 6, 2, 0, 7, 13, 15, 1, 13, 0, 12, 10, 15, 6, 2, 4, 14, 3},
		{5, 11, 14, 4, 0, 7, 12, 4, 4, 14, 12, 3, 4, 10, 7, 14, 6, 4, 14, 7, 0, 12, 5, 9, 15, 6, 15, 6, 3, 12, 0, 10, 11, 7, 1, 14, 13, 5, 1, 14, 5, 15, 12, 1, 9, 13, 9, 13, 14, 5, 10, 11, 12, 10, 15, 11, 9, 13, 2, 14, 9, 12, 2, 11},
		{2, 12, 5, 7, 1, 5, 2, 11, 8, 4, 15, 6, 9, 14, 5, 1, 15, 4, 3, 1, 11, 4, 2, 1, 4, 5, 4, 4, 7, 3, 3, 12, 4, 3, 2, 15, 13, 1, 14, 15, 1, 4, 6, 11, 13, 15, 6, 12, 12, 13, 6, 8, 10, 0, 10, 12, 1, 10, 3, 2, 9, 8, 2, 8},
		{10, 12, 12, 6, 8, 5, 4, 4, 5, 3, 6, 7, 15, 5, 10, 3, 8, 15, 14, 5, 6, 2, 14, 4, 1, 7, 1, 3, 12, 3, 12, 4, 10, 15, 6, 6, 0, 6, 6, 8, 6, 9, 5, 7, 5, 1, 9, 2, 4, 9, 0, 8, 1, 1, 14, 3, 7, 14, 8, 9, 0, 4, 11, 7},
		{13, 11, 14, 7, 0, 4, 0, 10, 12, 11, 10, 8, 6, 12, 13, 15, 9, 2, 14, 9, 3, 0, 12, 14, 11, 15, 4, 7, 15, 14, 4, 8, 15, 12, 9, 14, 7, 7, 9, 13, 14, 14, 4, 9, 13, 8, 1, 13, 6, 3, 12, 7, 0, 15, 6, 15, 7, 2, 3, 0, 9, 5, 13, 0},
		{3, 8, 12, 11, 5, 9, 9, 14, 8, 14, 14, 5, 9, 9, 12, 10, 3, 12, 13, 0, 0, 0, 6, 7, 12, 4, 2, 3, 8, 8, 9, 15, 11, 1, 12, 13, 10, 15, 11, 1, 2, 13, 10, 1, 7, 2, 7, 11, 8, 15, 7, 6, 4, 6, 5, 11, 11, 15, 2, 1, 11, 1, 1, 8},
		{10, 7, 7, 1, 4, 13, 9, 10, 2, 2, 3, 7, 12, 8, 5, 5, 5, 5, 3, 1, 5, 6, 8, 2, 8, 11, 5, 0, 4, 12, 12, 6, 7, 9, 14, 10, 11, 8, 0, 9, 11, 4, 14, 7, 7, 8, 2, 15, 12, 7, 4, 4, 13, 2, 0, 3, 14, 0, 1, 5, 2, 15, 7, 11},
		{3, 8, 10, 4, 1, 7, 3, 13, 5, 14, 0, 9, 3, 1, 0, 11, 2, 15, 4, 9, 6, 5, 14, 0, 2, 8, 1, 14, 7, 6, 1, 5, 5, 7, 2, 0, 5, 3, 4, 15, 13, 10, 9, 13, 13, 12, 5, 11, 11, 14, 13, 10, 8, 14, 0, 8, 1, 7, 2, 10, 12, 12, 1, 11},
		{11, 14, 4, 13, 3, 11, 10, 6, 15, 2, 5, 10, 14, 4, 13, 3, 12, 7, 12, 10, 4, 0, 0, 1, 14, 6, 1, 2, 2, 12, 9, 2, 3, 11, 1, 4, 10, 4, 4, 7, 7, 12, 4, 3, 12, 11, 9, 3, 15, 13, 6, 13, 7, 11, 5, 12, 5, 13, 15, 12, 0, 13, 12, 9},
		{8, 7, 2, 2, 5, 3, 10, 15, 10, 8, 1, 0, 4, 5, 7, 6, 15, 13, 2, 14, 6, 2, 9, 5, 9, 0, 5, 12, 8, 6, 4, 12, 6, 8, 14, 15, 7, 15, 11, 2, 2, 12, 7, 9, 7, 11, 15, 7, 0, 4, 5, 13, 7, 2, 5, 9, 0, 5, 7, 6, 7, 12, 4, 1},
		{11, 4, 2, 13, 6, 10, 9, 4, 12, 9, 9, 6, 4, 2, 14, 14, 9, 5, 5, 15, 15, 9, 8, 11, 4, 2, 8, 11, 14, 3, 8, 10, 14, 9, 6, 6, 4, 7, 11, 2, 3, 7, 5, 1, 14, 2, 9, 4, 0, 1, 10, 7, 6, 7, 1, 3, 13, 7, 3, 2, 12, 3, 6, 6},
		{11, 1, 14, 3, 14, 3, 9, 9, 0, 11, 14, 6, 14, 7, 14, 8, 4, 2, 5, 6, 13, 3, 4, 10, 8, 8, 10, 11, 5, 1, 15, 15, 7, 0, 4, 14, 15, 13, 14, 13, 3, 2, 1, 6, 0, 6, 6, 4, 15, 6, 0, 12, 5, 11, 1, 7, 3, 3, 13, 12, 12, 6, 3, 2},
		{7, 2, 10, 14, 14, 13, 4, 14, 10, 6, 0, 2, 7, 7, 2, 5, 14, 1, 5, 14, 15, 1, 2, 9, 2, 13, 1, 3, 6, 1, 3, 13, 10, 6, 11, 13, 1, 7, 13, 15, 2, 11, 9, 6, 13, 7, 9, 2, 3, 13, 10, 10, 6, 2, 5, 9, 1, 3, 0, 3, 1, 5, 3, 12},
		{11, 14, 4, 2, 10, 11, 15, 5, 9, 7, 8, 11, 10, 9, 5, 7, 14, 3, 12, 2, 7, 15, 12, 15, 4, 15, 12, 9, 2, 6, 6, 6, 8, 5, 0, 7, 14, 15, 14, 14, 3, 12, 7, 12, 2, 4, 1, 7, 1, 3, 4, 7, 1, 9, 11, 15, 15, 3, 7, 1, 10, 9, 14, 14},
		{4, 13, 11, 1, 9, 6, 5, 1, 11, 6, 6, 8, 3, 9, 8, 15, 13, 12, 3, 13, 5, 9, 10, 5, 12, 1, 15, 14, 12, 1, 10, 11, 5, 7, 3, 12, 9, 12, 0, 2, 2, 3, 14, 4, 2, 13, 1, 15, 11, 8, 3, 13, 0, 10, 5, 4, 6, 0, 14, 8, 1, 0, 6, 15},
		{15, 2, 0, 5, 2, 14, 9, 0, 10, 5, 12, 8, 5, 6, 0, 1, 9, 4, 4, 1, 4, 6, 14, 5, 3, 0, 2, 2, 14, 9, 7, 0, 2, 15, 12, 0, 10, 12, 9, 12, 15, 1, 9, 4, 15, 3, 0, 13, 0, 6, 5, 0, 2, 6, 11, 9, 13, 15, 6, 3, 5, 4, 0, 8},
		{4, 14, 8, 14, 13, 4, 4, 10, 6, 12, 15, 11, 7, 2, 15, 6, 9, 9, 1, 11, 13, 2, 7, 10, 4, 4, 5, 12, 14, 15, 8, 5, 6, 1, 11, 15, 4, 11, 5, 2, 5, 7, 3, 4, 5, 7, 3, 8, 10, 13, 7, 5, 6, 5, 10, 1, 12, 13, 3, 6, 2, 8, 7, 15},
		{3, 15, 4, 9, 14, 12, 6, 1, 7, 0, 7, 15, 10, 6, 5, 5, 15, 5, 9, 4, 7, 6, 14, 2, 1, 4, 10, 3, 12, 1, 7, 1, 0, 10, 2, 11, 14, 13, 7, 10, 5, 11, 5, 11, 15, 5, 0, 3, 15, 1, 2, 14, 13, 13, 10, 9, 15, 12, 10, 5, 2, 10, 0, 6},
		{4, 6, 5, 13, 11, 10, 15, 4, 2, 15, 13, 6, 7, 7, 4, 0, 4, 6, 7, 4, 9, 1, 6, 7, 6, 1, 4, 2, 0, 11, 6, 3, 14, 5, 9, 2, 2, 10, 1, 2, 13, 14, 4, 11, 4, 7, 12, 9, 8, 2, 2, 9, 5, 7, 9, 12, 8, 15, 0, 9, 12, 11, 1, 12},
		{11, 12, 11, 9, 8, 15, 4, 12, 13, 10, 6, 6, 6, 12, 3, 0, 6, 15, 15, 10, 6, 12, 5, 7, 10, 2, 7, 1, 6, 12, 9, 11, 11, 14, 1, 12, 15, 0, 6, 2, 12, 15, 4, 15, 14, 8, 3, 4, 15, 4, 13, 3, 14, 1, 3, 7, 6, 13, 9, 1, 0, 12, 4, 14},
		{12, 11, 13, 10, 10, 10, 3, 7, 12, 3, 13, 9, 6, 0, 12, 10, 4, 11, 5, 4, 11, 5, 7, 14, 6, 10, 12, 12, 13, 15, 12, 1, 13, 15, 15, 7, 1, 2, 8, 6, 1, 12, 12, 0, 4, 3, 3, 3, 7, 8, 9, 10, 7, 7, 0, 0, 11, 13, 15, 4, 9, 5, 10, 9},
		{6, 12, 3, 0, 9, 11, 6, 4, 9, 9, 1, 5, 9, 14, 3, 7, 15, 3, 5, 0, 5, 11, 7, 6, 13, 5, 10, 2, 12, 10, 2, 6, 0, 1, 1, 13, 9, 3, 11, 7, 8, 2, 10, 9, 13, 6, 6, 4, 12, 0, 3, 10, 9, 4, 15, 11, 14, 1, 9, 3, 0, 14, 6, 1},
		{11, 14, 10, 10, 11, 6, 4, 7, 10, 0, 7, 9, 3, 2, 13, 13, 9, 9, 2, 3, 3, 14, 10, 4, 14, 1, 10, 7, 14, 4, 9, 15, 3, 11, 5, 10, 7, 8, 3, 0, 1, 2, 2, 3, 12, 9, 6, 2, 11, 15, 3, 9, 3, 6, 8, 0, 4, 5, 7, 3, 0, 14, 7, 9},
		{4, 11, 13, 12, 6, 2, 3, 15, 15, 3, 5, 1, 0, 5, 10, 2, 5, 3, 7, 10, 15, 0, 5, 3, 2, 10, 12, 10, 8, 3, 9, 15, 5, 3, 7, 13, 5, 7, 13, 12, 5, 10, 2, 9, 10, 1, 9, 4, 14, 1, 10, 13, 1, 2, 2, 12, 5, 3, 14, 7, 7, 8, 13, 13},
		{10, 12, 11, 10, 0, 15, 4, 3, 0, 8, 3, 0, 15, 0, 3, 10, 10, 9, 15, 3, 13, 3, 8, 3, 8, 2, 14, 7, 1, 6, 13, 8, 2, 2, 12, 3, 3, 0, 10, 12, 0, 1, 1, 7, 5, 0, 13, 10, 7, 13, 9, 9, 13, 7, 0, 1, 0, 2, 14, 2, 13, 0, 8, 3},
		{11, 3, 11, 10, 12, 15, 11, 6, 14, 8, 8, 5, 7, 11, 3, 1, 13, 7, 13, 4, 15, 7, 2, 3, 8, 7, 3, 8, 9, 15, 10, 15, 9, 0, 5, 4, 1, 7, 13, 8, 2, 7, 1, 10, 1, 12, 12, 1, 7, 12, 13, 5, 14, 10, 9, 15, 12, 2, 10, 3, 10, 3, 9, 12},
		{9, 8, 11, 0, 5, 6, 1, 5, 9, 1, 0, 12, 12, 0, 12, 11, 2, 8, 4, 0, 1, 7, 7, 5, 1, 14, 1, 9, 13, 7, 2, 12, 8, 9, 12, 13, 1, 11, 5, 3, 12, 14, 15, 4, 9, 8, 12, 7, 11, 1, 3, 9, 11, 5, 7, 14, 4, 6, 12, 3, 4, 12, 7, 9},
		{10, 12, 2, 14, 14, 1, 11, 8, 3, 7, 13, 7, 2, 1, 14, 13, 7, 6, 15, 8, 15, 12, 13, 10, 11, 15, 4, 2, 6, 13, 12, 3, 2, 10, 15, 14, 10, 11, 8, 14, 9, 3, 12, 9, 15, 2, 14, 14, 5, 13, 7, 6, 2, 1, 1, 4, 1, 0, 13, 10, 1, 0, 2, 9},
		{10, 5, 11, 14, 12, 1, 12, 7, 12, 8, 10, 5, 6, 10, 0, 7, 5, 6, 11, 11, 13, 12, 0, 13, 0, 6, 11, 0, 14, 4, 2, 1, 12, 7, 1, 10, 7, 15, 5, 3, 14, 15, 1, 3, 1, 2, 10, 4, 11, 8, 2, 11, 2, 5, 5, 4, 15, 5, 10, 3, 1, 7, 2, 14},
	}
	mat := GenerateMatrix(hash)

	if *mat != expectedMatrix {
		t.Fatal("The generated matrix doesn't match the test vector")
	}
}

func TestMatrix_HeavyHash(t *testing.T) {
	expected, err := hex.DecodeString("f32d27ec12b058b602252dc79dbd5c32958ef5c153731dfc8220d25b58ffa691")
	if err != nil {
		t.Fatal(err)
	}
	input := []byte{0xC1, 0xEC, 0xFD, 0xFC}
	writer := hashes.PoWHashWriter()
	writer.InfallibleWrite(input)
	hashed := testMatrix.kHeavyHash(writer.Finalize())

	if !bytes.Equal(expected, hashed.ByteSlice()) {
		t.Fatalf("expected: %x == %s", expected, hashed)
	}

}

var testMatrix = matrix{
	{13, 2, 14, 13, 2, 15, 14, 3, 10, 4, 1, 8, 4, 3, 8, 15, 15, 15, 15, 15, 2, 11, 15, 15, 15, 1, 7, 12, 12, 4, 2, 0, 6, 1, 14, 10, 12, 14, 15, 8, 10, 12, 0, 5, 13, 3, 14, 10, 10, 6, 12, 11, 11, 7, 6, 6, 10, 2, 2, 4, 11, 12, 0, 5},
	{4, 13, 0, 2, 1, 15, 13, 13, 11, 2, 5, 12, 15, 7, 0, 10, 7, 2, 6, 3, 12, 0, 12, 0, 2, 6, 7, 7, 7, 7, 10, 12, 11, 14, 12, 12, 4, 11, 10, 0, 10, 11, 2, 10, 1, 7, 7, 12, 15, 9, 5, 14, 9, 12, 3, 0, 12, 13, 4, 13, 8, 15, 11, 6},
	{14, 6, 15, 9, 8, 2, 2, 12, 2, 3, 4, 12, 13, 15, 4, 5, 13, 4, 3, 0, 14, 3, 5, 14, 3, 13, 4, 15, 9, 12, 7, 15, 5, 1, 13, 12, 9, 9, 8, 11, 14, 11, 4, 10, 12, 6, 12, 8, 6, 3, 9, 8, 1, 6, 0, 5, 8, 9, 12, 5, 14, 15, 2, 2},
	{9, 6, 7, 6, 0, 11, 5, 6, 2, 14, 12, 6, 4, 13, 8, 9, 2, 1, 9, 7, 4, 5, 10, 8, 11, 11, 11, 15, 7, 11, 1, 14, 3, 8, 14, 8, 2, 8, 13, 7, 8, 8, 15, 7, 1, 13, 7, 9, 1, 7, 15, 15, 0, 0, 12, 15, 13, 5, 13, 10, 1, 5, 6, 13},
	{4, 0, 12, 10, 6, 11, 14, 2, 2, 15, 4, 1, 2, 4, 2, 12, 13, 1, 9, 10, 8, 0, 2, 10, 13, 8, 9, 7, 5, 3, 8, 2, 6, 6, 1, 12, 3, 0, 1, 4, 2, 8, 3, 13, 6, 15, 0, 13, 14, 4, 15, 0, 7, 3, 7, 8, 5, 14, 14, 5, 5, 0, 1, 2},
	{12, 14, 6, 3, 3, 4, 6, 7, 1, 3, 2, 7, 15, 15, 15, 10, 9, 12, 0, 6, 3, 8, 5, 0, 13, 5, 0, 6, 0, 14, 2, 12, 10, 4, 11, 2, 10, 7, 7, 6, 8, 11, 4, 4, 11, 9, 3, 12, 10, 5, 2, 6, 5, 5, 10, 13, 12, 10, 1, 6, 14, 7, 12, 4},
	{7, 14, 6, 7, 7, 12, 4, 1, 8, 6, 8, 13, 13, 5, 12, 14, 10, 8, 6, 2, 12, 3, 8, 15, 5, 15, 15, 3, 14, 0, 8, 6, 9, 12, 9, 7, 3, 8, 4, 0, 7, 14, 3, 3, 13, 14, 3, 7, 3, 2, 2, 3, 3, 12, 6, 7, 4, 1, 14, 10, 6, 10, 2, 9},
	{14, 11, 15, 5, 7, 10, 1, 11, 4, 2, 6, 2, 9, 7, 4, 0, 9, 12, 11, 2, 3, 13, 1, 5, 4, 10, 5, 6, 6, 12, 8, 1, 1, 15, 4, 2, 12, 12, 0, 4, 14, 3, 11, 1, 7, 5, 9, 4, 3, 15, 7, 3, 15, 9, 8, 3, 8, 3, 3, 6, 7, 6, 9, 2},
	{10, 4, 6, 10, 5, 2, 15, 12, 0, 14, 14, 15, 14, 0, 12, 9, 1, 12, 4, 5, 5, 2, 10, 4, 2, 13, 11, 3, 1, 8, 10, 0, 7, 0, 12, 4, 11, 1, 14, 6, 14, 5, 5, 11, 11, 1, 3, 8, 0, 6, 11, 11, 8, 4, 7, 6, 14, 4, 9, 14, 9, 7, 13, 9},
	{12, 7, 9, 8, 2, 3, 3, 5, 14, 8, 0, 9, 7, 4, 2, 15, 15, 3, 11, 11, 8, 5, 7, 5, 0, 15, 10, 8, 0, 13, 1, 14, 8, 10, 1, 4, 13, 1, 13, 3, 11, 11, 2, 3, 10, 6, 8, 14, 15, 2, 10, 10, 12, 7, 7, 6, 6, 3, 13, 8, 1, 14, 2, 1},
	{2, 11, 6, 9, 13, 3, 12, 6, 0, 4, 6, 13, 8, 14, 6, 9, 10, 2, 10, 8, 4, 13, 6, 5, 0, 13, 15, 4, 2, 2, 1, 7, 5, 3, 3, 13, 7, 3, 5, 9, 15, 14, 14, 6, 0, 15, 11, 2, 4, 15, 6, 9, 8, 9, 15, 2, 6, 9, 15, 8, 4, 4, 11, 1},
	{10, 11, 8, 3, 11, 13, 10, 2, 2, 5, 2, 14, 15, 10, 2, 11, 0, 1, 8, 2, 14, 1, 10, 0, 3, 7, 5, 10, 7, 8, 15, 7, 2, 5, 13, 4, 10, 3, 6, 2, 3, 9, 6, 11, 7, 14, 1, 11, 9, 3, 3, 7, 6, 0, 9, 11, 4, 10, 4, 1, 9, 7, 4, 15},
	{13, 8, 15, 14, 11, 12, 5, 3, 9, 14, 1, 5, 14, 13, 14, 5, 13, 5, 4, 10, 9, 9, 0, 0, 6, 12, 5, 7, 2, 7, 2, 6, 6, 6, 1, 12, 9, 15, 7, 11, 11, 10, 11, 1, 10, 10, 0, 8, 1, 4, 5, 5, 8, 10, 10, 15, 6, 8, 13, 11, 11, 3, 15, 5},
	{8, 11, 5, 10, 1, 10, 9, 1, 12, 7, 6, 11, 1, 1, 4, 1, 2, 8, 4, 4, 7, 7, 8, 2, 7, 1, 14, 1, 8, 15, 15, 12, 10, 4, 15, 11, 3, 6, 10, 7, 4, 0, 10, 9, 11, 7, 1, 14, 4, 14, 3, 14, 10, 4, 13, 12, 5, 3, 12, 7, 10, 8, 0, 3},
	{9, 11, 6, 15, 14, 10, 0, 4, 7, 7, 6, 0, 7, 7, 12, 15, 5, 4, 12, 3, 7, 3, 0, 12, 2, 7, 11, 6, 7, 3, 2, 8, 5, 11, 9, 4, 3, 8, 11, 12, 3, 5, 14, 12, 4, 13, 12, 0, 3, 14, 4, 9, 1, 1, 9, 14, 10, 14, 8, 15, 6, 14, 10, 15},
	{10, 14, 10, 0, 10, 12, 15, 0, 3, 9, 11, 10, 3, 5, 1, 1, 9, 1, 7, 15, 7, 8, 10, 10, 12, 11, 5, 1, 10, 3, 6, 6, 13, 0, 13, 1, 4, 5, 9, 4, 9, 15, 8, 4, 13, 13, 4, 5, 5, 11, 1, 13, 15, 3, 10, 15, 7, 11, 10, 15, 8, 12, 10, 3},
	{8, 5, 11, 3, 8, 13, 15, 15, 3, 12, 1, 13, 1, 7, 1, 5, 6, 13, 7, 8, 5, 1, 12, 3, 10, 7, 12, 6, 14, 12, 15, 5, 3, 12, 2, 15, 11, 13, 1, 13, 8, 5, 8, 0, 13, 15, 7, 13, 6, 13, 10, 1, 11, 0, 8, 9, 5, 11, 2, 9, 9, 10, 4, 15},
	{0, 4, 12, 14, 3, 1, 7, 5, 11, 13, 5, 3, 11, 12, 6, 8, 10, 15, 11, 8, 7, 10, 0, 2, 5, 15, 6, 10, 4, 2, 3, 1, 13, 7, 6, 12, 14, 7, 6, 14, 12, 10, 6, 14, 12, 0, 12, 11, 6, 9, 3, 1, 12, 15, 15, 3, 5, 5, 10, 11, 7, 15, 13, 3},
	{12, 14, 2, 14, 13, 6, 15, 7, 8, 8, 14, 13, 9, 2, 2, 10, 3, 15, 6, 10, 11, 7, 13, 0, 12, 1, 5, 8, 8, 12, 1, 11, 1, 3, 2, 4, 10, 7, 7, 7, 3, 10, 7, 2, 2, 3, 0, 1, 13, 5, 8, 2, 14, 0, 11, 13, 9, 3, 13, 2, 14, 2, 15, 4},
	{0, 0, 13, 6, 9, 12, 15, 7, 8, 0, 7, 4, 12, 15, 3, 2, 7, 1, 14, 4, 9, 3, 13, 12, 11, 12, 9, 9, 3, 7, 10, 9, 1, 9, 10, 2, 10, 14, 11, 0, 14, 4, 15, 12, 12, 9, 9, 8, 14, 1, 9, 14, 0, 6, 1, 0, 13, 9, 7, 6, 13, 2, 3, 9},
	{8, 0, 10, 13, 0, 7, 9, 7, 5, 1, 0, 3, 7, 10, 3, 15, 1, 15, 3, 11, 2, 6, 3, 10, 0, 10, 10, 3, 4, 15, 8, 6, 11, 11, 7, 5, 8, 5, 7, 15, 1, 11, 7, 13, 13, 6, 13, 13, 4, 2, 3, 15, 9, 5, 10, 6, 6, 6, 3, 11, 15, 13, 1, 15},
	{1, 1, 2, 10, 2, 2, 9, 5, 9, 2, 0, 1, 14, 2, 11, 6, 11, 6, 1, 0, 13, 7, 14, 1, 15, 14, 13, 7, 12, 11, 8, 11, 2, 11, 6, 10, 2, 3, 0, 0, 15, 0, 4, 6, 4, 12, 5, 5, 7, 14, 10, 6, 0, 3, 13, 0, 8, 1, 13, 10, 5, 1, 7, 5},
	{0, 5, 2, 12, 10, 2, 5, 1, 14, 0, 1, 4, 15, 11, 8, 7, 11, 14, 15, 6, 4, 1, 6, 6, 7, 13, 12, 5, 13, 2, 1, 6, 2, 13, 5, 15, 0, 8, 8, 6, 5, 5, 2, 0, 3, 13, 14, 2, 10, 5, 7, 6, 14, 5, 1, 4, 11, 2, 11, 1, 8, 15, 2, 4},
	{9, 9, 4, 5, 2, 5, 3, 12, 14, 5, 1, 3, 3, 0, 0, 6, 7, 14, 0, 15, 14, 11, 3, 10, 1, 9, 4, 14, 7, 14, 1, 0, 15, 11, 5, 9, 4, 0, 0, 10, 4, 4, 0, 7, 8, 15, 12, 8, 10, 8, 1, 2, 1, 11, 12, 14, 14, 14, 8, 10, 1, 5, 13, 10},
	{5, 10, 4, 4, 11, 10, 0, 6, 0, 12, 10, 5, 9, 11, 8, 10, 11, 3, 11, 14, 12, 9, 4, 6, 11, 12, 8, 7, 6, 14, 0, 6, 12, 4, 5, 3, 9, 0, 11, 6, 1, 3, 2, 12, 8, 9, 7, 12, 14, 7, 12, 6, 11, 13, 0, 2, 1, 3, 1, 8, 12, 2, 15, 15},
	{10, 11, 2, 3, 11, 10, 1, 7, 1, 10, 10, 14, 5, 13, 10, 3, 11, 15, 9, 14, 11, 11, 3, 15, 11, 6, 15, 13, 13, 1, 1, 10, 5, 1, 5, 11, 10, 3, 9, 12, 12, 1, 5, 6, 3, 3, 1, 1, 12, 8, 3, 15, 6, 2, 8, 14, 3, 4, 10, 9, 7, 13, 2, 6},
	{12, 0, 1, 0, 4, 3, 3, 6, 8, 3, 1, 13, 6, 12, 1, 1, 1, 4, 12, 4, 4, 9, 9, 14, 15, 3, 6, 4, 11, 1, 12, 5, 6, 0, 10, 9, 1, 8, 14, 5, 2, 8, 4, 15, 12, 13, 7, 14, 12, 2, 6, 9, 4, 13, 0, 15, 10, 10, 6, 12, 7, 12, 9, 10},
	{0, 8, 5, 11, 12, 12, 11, 7, 2, 9, 2, 15, 1, 1, 0, 0, 6, 5, 10, 1, 11, 12, 8, 7, 1, 7, 10, 4, 2, 8, 2, 5, 1, 1, 2, 9, 2, 0, 3, 7, 5, 1, 5, 5, 3, 1, 4, 3, 14, 8, 11, 7, 8, 0, 2, 13, 3, 15, 1, 13, 14, 15, 11, 13},
	{8, 13, 5, 14, 2, 9, 9, 13, 15, 8, 2, 14, 4, 2, 6, 0, 1, 13, 10, 13, 6, 12, 15, 11, 6, 11, 9, 9, 2, 9, 6, 14, 2, 9, 12, 1, 13, 9, 5, 11, 10, 4, 4, 5, 8, 9, 13, 10, 9, 0, 5, 15, 4, 12, 7, 10, 6, 5, 5, 15, 8, 8, 11, 14},
	{6, 9, 6, 7, 1, 15, 0, 1, 4, 15, 5, 3, 10, 9, 15, 9, 14, 12, 7, 6, 3, 0, 12, 8, 12, 2, 11, 8, 11, 8, 1, 10, 10, 7, 7, 5, 3, 5, 1, 2, 13, 11, 2, 5, 2, 10, 10, 1, 14, 14, 8, 1, 11, 1, 2, 6, 15, 10, 8, 7, 10, 7, 0, 3},
	{12, 6, 11, 1, 1, 7, 8, 1, 5, 5, 8, 4, 6, 5, 6, 4, 2, 8, 4, 1, 0, 0, 14, 2, 10, 14, 14, 11, 2, 9, 14, 15, 12, 14, 9, 3, 7, 14, 4, 7, 12, 9, 3, 5, 1, 0, 12, 9, 10, 5, 11, 12, 10, 10, 6, 14, 6, 13, 13, 5, 5, 10, 13, 10},
	{12, 6, 13, 0, 8, 0, 10, 6, 15, 15, 7, 3, 0, 10, 13, 14, 10, 13, 5, 13, 15, 14, 3, 4, 10, 10, 9, 6, 6, 15, 2, 7, 0, 10, 6, 14, 2, 9, 11, 7, 5, 5, 13, 14, 11, 15, 9, 4, 2, 0, 15, 5, 4, 14, 14, 1, 3, 4, 5, 8, 1, 1, 10, 12},
	{2, 5, 0, 4, 11, 5, 5, 6, 10, 4, 6, 7, 10, 3, 0, 14, 14, 0, 12, 15, 11, 12, 13, 7, 6, 3, 9, 1, 9, 8, 8, 8, 4, 10, 3, 1, 7, 10, 3, 2, 12, 6, 15, 14, 0, 6, 8, 10, 1, 9, 12, 12, 15, 7, 1, 11, 15, 13, 0, 4, 10, 0, 12, 11},
	{8, 12, 14, 15, 14, 15, 10, 0, 2, 14, 3, 1, 2, 6, 0, 2, 1, 7, 9, 0, 15, 13, 5, 14, 6, 8, 15, 4, 15, 6, 10, 6, 15, 3, 12, 8, 5, 4, 10, 5, 3, 0, 4, 13, 10, 9, 8, 4, 6, 3, 9, 6, 12, 11, 9, 13, 8, 10, 9, 9, 8, 12, 1, 2},
	{11, 10, 15, 15, 5, 14, 15, 7, 5, 9, 14, 14, 7, 11, 6, 6, 3, 8, 2, 3, 4, 14, 11, 1, 12, 15, 11, 6, 0, 0, 13, 7, 14, 3, 12, 14, 0, 15, 6, 1, 11, 2, 11, 8, 3, 13, 4, 12, 10, 13, 7, 14, 9, 13, 3, 10, 2, 14, 13, 4, 12, 13, 14, 10},
	{1, 11, 2, 12, 1, 10, 7, 12, 3, 3, 14, 9, 1, 10, 0, 11, 8, 10, 12, 12, 4, 12, 2, 11, 5, 0, 3, 15, 8, 2, 14, 3, 10, 2, 1, 13, 6, 14, 0, 0, 8, 11, 6, 13, 15, 10, 12, 7, 7, 11, 14, 9, 2, 7, 6, 8, 14, 9, 14, 10, 11, 9, 9, 12},
	{5, 10, 14, 2, 1, 4, 11, 5, 10, 2, 13, 9, 6, 12, 11, 5, 13, 4, 5, 14, 8, 7, 15, 9, 8, 4, 5, 2, 9, 11, 5, 3, 12, 2, 6, 1, 7, 4, 11, 4, 15, 0, 5, 2, 13, 11, 11, 2, 15, 10, 0, 12, 5, 8, 10, 1, 4, 11, 3, 13, 11, 7, 9, 14},
	{9, 8, 10, 5, 0, 2, 5, 8, 7, 3, 3, 6, 11, 1, 13, 15, 4, 4, 11, 6, 2, 6, 13, 11, 2, 6, 9, 4, 5, 13, 12, 2, 8, 7, 7, 12, 14, 15, 5, 12, 7, 0, 15, 15, 0, 5, 15, 0, 3, 9, 10, 15, 9, 11, 10, 10, 5, 3, 9, 3, 12, 13, 0, 13},
	{1, 11, 15, 0, 10, 5, 3, 5, 6, 7, 1, 11, 4, 11, 4, 2, 5, 12, 2, 5, 5, 6, 1, 5, 14, 9, 1, 5, 14, 12, 6, 10, 0, 8, 5, 11, 11, 11, 12, 10, 8, 10, 10, 1, 14, 1, 0, 8, 4, 7, 0, 11, 3, 1, 11, 12, 11, 8, 14, 15, 9, 3, 1, 14},
	{14, 11, 12, 12, 4, 6, 8, 14, 15, 1, 11, 2, 13, 3, 6, 2, 7, 1, 8, 1, 4, 9, 11, 15, 8, 1, 10, 13, 4, 13, 2, 7, 7, 10, 5, 2, 12, 12, 12, 3, 10, 8, 2, 11, 0, 3, 8, 9, 4, 2, 15, 7, 15, 6, 4, 6, 12, 7, 14, 9, 9, 8, 14, 12},
	{15, 4, 8, 12, 11, 11, 9, 5, 0, 0, 7, 6, 10, 5, 8, 2, 5, 6, 14, 11, 13, 0, 13, 15, 5, 4, 9, 15, 13, 12, 14, 15, 10, 2, 3, 6, 10, 14, 1, 8, 6, 7, 10, 1, 14, 9, 12, 13, 7, 2, 12, 10, 6, 11, 15, 1, 15, 11, 13, 0, 6, 13, 7, 15},
	{3, 3, 12, 5, 14, 9, 14, 14, 8, 0, 9, 1, 2, 2, 14, 11, 7, 1, 3, 1, 14, 15, 12, 8, 14, 2, 4, 13, 10, 5, 10, 8, 1, 7, 6, 5, 4, 2, 11, 5, 4, 13, 14, 6, 13, 15, 6, 6, 7, 12, 11, 5, 13, 10, 9, 13, 9, 14, 5, 6, 7, 14, 11, 7},
	{14, 12, 11, 5, 0, 5, 10, 5, 7, 1, 7, 11, 1, 0, 13, 6, 5, 14, 3, 0, 5, 14, 6, 7, 8, 5, 8, 6, 6, 3, 6, 1, 8, 3, 10, 7, 15, 6, 11, 6, 6, 7, 13, 2, 2, 0, 0, 11, 1, 15, 2, 14, 5, 1, 4, 8, 0, 1, 8, 0, 1, 1, 2, 2},
	{10, 13, 13, 3, 15, 14, 9, 12, 15, 15, 8, 5, 8, 10, 5, 9, 6, 6, 7, 15, 1, 0, 14, 9, 1, 11, 6, 11, 13, 4, 6, 14, 9, 12, 13, 8, 14, 6, 14, 2, 3, 15, 4, 4, 14, 4, 9, 12, 8, 0, 9, 11, 13, 10, 8, 14, 3, 5, 7, 11, 6, 7, 15, 2},
	{9, 9, 11, 6, 11, 0, 5, 4, 8, 10, 8, 11, 2, 12, 8, 7, 11, 13, 6, 1, 13, 13, 11, 4, 5, 7, 7, 9, 6, 4, 12, 0, 11, 8, 6, 12, 11, 4, 15, 11, 12, 8, 11, 11, 1, 3, 6, 14, 9, 6, 7, 5, 0, 10, 3, 15, 13, 7, 0, 1, 13, 15, 1, 14},
	{10, 6, 8, 7, 3, 6, 9, 15, 1, 3, 10, 14, 9, 0, 0, 10, 0, 15, 2, 0, 0, 0, 6, 0, 13, 9, 9, 1, 8, 6, 13, 2, 1, 9, 14, 9, 1, 4, 8, 4, 2, 0, 8, 5, 0, 11, 12, 15, 13, 1, 14, 14, 15, 7, 8, 4, 4, 12, 1, 12, 8, 3, 9, 5},
	{12, 11, 1, 4, 10, 14, 8, 12, 2, 4, 15, 2, 9, 7, 7, 11, 15, 12, 10, 11, 7, 4, 13, 0, 8, 6, 8, 8, 10, 5, 5, 13, 3, 7, 9, 13, 13, 14, 6, 8, 1, 5, 7, 12, 4, 4, 6, 9, 13, 1, 6, 1, 6, 14, 5, 8, 2, 10, 4, 10, 1, 9, 6, 15},
	{4, 13, 4, 9, 6, 11, 1, 8, 7, 11, 11, 1, 3, 10, 12, 11, 1, 10, 6, 10, 0, 7, 3, 0, 0, 6, 3, 9, 2, 1, 4, 8, 2, 10, 2, 15, 9, 15, 14, 14, 15, 14, 3, 2, 7, 6, 6, 10, 8, 8, 4, 11, 1, 13, 6, 0, 2, 10, 0, 11, 15, 14, 6, 9},
	{15, 0, 12, 13, 0, 9, 10, 4, 11, 5, 10, 0, 8, 7, 3, 2, 12, 6, 3, 8, 5, 15, 14, 2, 13, 13, 6, 11, 5, 6, 9, 10, 14, 5, 14, 4, 9, 7, 5, 11, 13, 2, 7, 1, 14, 9, 0, 7, 8, 12, 11, 15, 2, 1, 5, 11, 3, 7, 5, 1, 6, 3, 8, 6},
	{0, 3, 8, 1, 4, 6, 3, 1, 3, 8, 2, 0, 15, 15, 14, 15, 13, 10, 11, 9, 2, 11, 5, 12, 3, 3, 0, 1, 5, 3, 11, 6, 10, 11, 8, 5, 7, 15, 4, 12, 8, 8, 12, 12, 12, 1, 9, 4, 11, 6, 10, 11, 1, 12, 8, 12, 5, 6, 1, 14, 2, 10, 3, 0},
	{10, 13, 6, 9, 11, 1, 4, 10, 0, 13, 8, 7, 4, 12, 15, 5, 14, 12, 6, 9, 0, 0, 10, 5, 13, 10, 15, 3, 0, 8, 7, 0, 9, 8, 10, 6, 11, 8, 10, 13, 11, 7, 5, 5, 9, 13, 1, 15, 0, 5, 15, 5, 4, 7, 9, 9, 15, 8, 2, 6, 3, 8, 5, 8},
	{14, 0, 6, 2, 4, 12, 2, 13, 6, 10, 5, 2, 2, 1, 6, 11, 1, 6, 9, 13, 0, 13, 9, 3, 12, 4, 3, 8, 7, 0, 9, 12, 0, 1, 7, 10, 10, 7, 3, 9, 13, 5, 15, 4, 13, 0, 8, 5, 4, 14, 11, 3, 3, 13, 15, 9, 9, 12, 9, 5, 2, 0, 1, 14},
	{4, 14, 13, 0, 14, 15, 11, 10, 11, 1, 3, 3, 9, 1, 12, 8, 6, 5, 15, 11, 1, 7, 5, 3, 8, 13, 0, 13, 11, 5, 8, 1, 8, 6, 13, 4, 13, 7, 12, 6, 5, 5, 7, 0, 12, 1, 1, 8, 1, 6, 4, 2, 8, 8, 15, 11, 11, 11, 4, 4, 4, 7, 13, 12},
	{14, 15, 10, 0, 4, 3, 1, 9, 13, 7, 9, 9, 15, 5, 0, 3, 9, 6, 4, 7, 13, 11, 3, 2, 7, 1, 6, 8, 13, 7, 10, 4, 3, 9, 5, 9, 2, 6, 10, 7, 9, 13, 2, 14, 2, 14, 7, 2, 14, 2, 8, 8, 0, 9, 0, 9, 12, 6, 7, 7, 6, 8, 12, 13},
	{5, 15, 8, 12, 11, 3, 13, 4, 5, 14, 10, 4, 15, 15, 1, 10, 9, 14, 6, 6, 4, 12, 4, 9, 12, 2, 15, 13, 2, 5, 12, 2, 3, 2, 15, 11, 12, 2, 6, 2, 11, 6, 7, 9, 12, 10, 5, 1, 1, 5, 9, 6, 14, 11, 3, 11, 6, 10, 11, 11, 0, 12, 15, 1},
	{12, 6, 8, 10, 2, 5, 7, 9, 8, 14, 15, 15, 13, 10, 15, 3, 10, 10, 6, 10, 14, 10, 7, 5, 3, 7, 6, 12, 11, 12, 8, 9, 12, 9, 15, 15, 15, 7, 8, 3, 15, 14, 1, 12, 0, 0, 4, 0, 9, 10, 8, 7, 14, 10, 8, 14, 6, 2, 8, 1, 11, 10, 0, 1},
	{12, 1, 2, 12, 7, 10, 4, 11, 5, 14, 10, 2, 2, 9, 4, 13, 3, 14, 3, 15, 5, 0, 14, 7, 7, 15, 6, 5, 2, 8, 15, 9, 6, 6, 13, 10, 9, 8, 6, 3, 14, 7, 12, 9, 7, 8, 13, 12, 14, 13, 6, 0, 5, 1, 9, 12, 14, 0, 11, 11, 6, 3, 11, 7},
	{15, 4, 8, 12, 8, 11, 4, 15, 1, 6, 2, 13, 1, 7, 7, 12, 0, 8, 14, 14, 10, 14, 0, 12, 0, 3, 3, 11, 7, 4, 2, 13, 0, 0, 11, 2, 5, 8, 12, 11, 6, 5, 6, 0, 0, 4, 0, 0, 1, 9, 9, 11, 3, 2, 13, 4, 13, 9, 15, 4, 7, 8, 3, 2},
	{3, 13, 8, 8, 12, 10, 5, 4, 7, 13, 10, 13, 14, 3, 2, 12, 11, 0, 9, 5, 6, 4, 14, 4, 6, 9, 2, 5, 10, 3, 9, 10, 5, 0, 12, 5, 15, 5, 15, 15, 2, 12, 3, 11, 0, 15, 9, 14, 1, 5, 6, 6, 14, 5, 8, 0, 5, 9, 3, 7, 7, 12, 15, 1},
	{1, 11, 7, 4, 13, 3, 0, 8, 11, 9, 15, 1, 4, 12, 2, 12, 10, 4, 14, 3, 9, 14, 14, 2, 3, 11, 12, 4, 5, 10, 6, 15, 2, 13, 13, 9, 9, 1, 11, 12, 12, 14, 1, 5, 15, 1, 7, 14, 12, 10, 11, 13, 13, 5, 2, 4, 7, 7, 9, 4, 14, 15, 13, 10},
	{14, 15, 9, 14, 9, 5, 13, 2, 0, 0, 14, 8, 6, 2, 0, 7, 11, 10, 2, 13, 2, 14, 9, 6, 4, 11, 5, 14, 6, 1, 6, 14, 6, 3, 9, 5, 2, 9, 3, 11, 1, 14, 5, 4, 12, 5, 3, 5, 11, 3, 11, 6, 13, 7, 13, 7, 4, 9, 4, 13, 8, 3, 5, 11},
	{13, 12, 12, 13, 8, 2, 4, 2, 10, 6, 3, 5, 7, 7, 6, 13, 8, 6, 15, 4, 12, 7, 15, 4, 3, 9, 8, 15, 0, 3, 12, 1, 9, 8, 13, 10, 15, 4, 14, 1, 6, 15, 0, 4, 8, 9, 3, 1, 3, 15, 5, 5, 1, 11, 11, 10, 11, 10, 8, 8, 5, 4, 13, 0},
	{8, 4, 15, 9, 14, 9, 5, 8, 8, 10, 5, 15, 9, 8, 12, 5, 11, 10, 2, 12, 13, 1, 0, 2, 6, 13, 11, 9, 12, 0, 5, 0, 11, 5, 14, 12, 3, 4, 2, 10, 3, 12, 5, 15, 4, 8, 14, 1, 0, 13, 9, 5, 2, 4, 13, 8, 2, 5, 8, 9, 15, 3, 5, 5},
	{0, 3, 3, 4, 6, 5, 5, 1, 3, 2, 14, 5, 10, 7, 15, 11, 7, 13, 15, 4, 0, 12, 9, 15, 12, 0, 3, 1, 14, 1, 12, 9, 13, 8, 9, 15, 12, 3, 5, 11, 3, 11, 4, 1, 9, 4, 13, 7, 4, 10, 6, 14, 13, 0, 9, 11, 15, 15, 3, 3, 13, 15, 10, 15},
}
