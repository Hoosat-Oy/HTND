package pow

import (
	// "github.com/chewxy/math"

	"math"

	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/hashes"
)

const eps float64 = 1e-9

// type matrix [64][64]uint16
type matrix [64][64]uint16
type floatMatrix [64][64]float64

// func generateMatrix(hash *externalapi.DomainHash) *matrix {
// 	var mat matrix
// 	generator := newxoShiRo256PlusPlus(hash)
// 	for {
// 		for i := range mat {
// 			for j := 0; j < 64; j += 16 {
// 				val := generator.Uint64()
// 				for shift := 0; shift < 16; shift++ {
// 					mat[i][j+shift] = uint16(val >> (4 * shift) & 0x0F)
// 				}
// 			}
// 		}
// 		if mat.computeRank() == 64 {
// 			return &mat
// 		}
// 	}
// }

func GenerateMatrix(hash *externalapi.DomainHash) *matrix {
	var mat matrix
	generator := newxoShiRo256PlusPlus(hash)

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
		rank := mat.computeRank()
		if rank == 64 {
			return &mat
		}
	}
}

func GenerateHoohashMatrix(hash *externalapi.DomainHash) *matrix {
	var mat matrix
	generator := newxoShiRo256PlusPlus(hash)

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

func GenerateHoohashMatrix102(hash *externalapi.DomainHash) *floatMatrix {
	var mat floatMatrix
	generator := newxoShiRo256PlusPlus(hash)
	const normalize float64 = 100000000

	for i := 0; i < 64; i++ {
		for j := 0; j < 64; j++ {
			val := generator.Uint64()
			lower4Bytes := uint32(val & 0xFFFFFFFF)
			matrixVal := float64(lower4Bytes)/float64(math.MaxUint32)*(normalize*2) - normalize
			mat[i][j] = matrixVal
		}
	}
	return &mat
}

// func generateMatrix(hash *externalapi.DomainHash) *matrix {
// 	var mat matrix
// 	generator := newxoShiRo256PlusPlus(hash)

// 	for {
// 		for i := range mat {
// 			for j := 0; j < 128; j += 16 {
// 				val := generator.Uint64()
// 				for shift := 0; shift < 16; shift++ {
// 					mat[i][j+shift] = uint16(val >> (4 * shift) & 0x0F)
// 				}
// 			}
// 		}
// 		if mat.computeRank() == 128 {
// 			return &mat
// 		}
// 	}
// }

// Basic Non-linear Operations are fast but less computationally intensive.
// Intermediate Non-linear Operations increase complexity with additional trigonometric functions.
// Advanced Non-linear Operations involve more complex combinations of trigonometric, exponential, and logarithmic functions.
// Very Complex Non-linear Operations introduce even more layers of computation, involving multiple transcendental functions.
// Extremely Complex Non-linear Operations are the most computationally intensive, combining high-power terms, exponentials, and logarithms of absolute values.

func BasicComplexNonLinear(x float64) float64 {
	return math.Sin(x) + math.Cos(x)
}

func MediumComplexNonLinear(x float64) float64 {
	return math.Exp(math.Sin(x) + math.Cos(x))
}

func IntermediateComplexNonLinear(x float64) float64 {
	if x == math.Pi/2 || x == 3*math.Pi/2 {
		return 0 // Avoid singularity
	}
	return math.Sin(x) * math.Cos(x) * math.Tan(x)
}

func AdvancedComplexNonLinear(x float64) float64 {
	if x <= -1 {
		return 0 // Avoid log domain error
	}
	return math.Exp(math.Sin(x)+math.Cos(x*x)) * math.Log1p(x*x+1)
}

func HighComplexNonLinear(x float64) float64 {
	return math.Exp(x) * math.Log(x+1)
}

func VeryComplexNonLinear(x float64) float64 {
	if x == math.Pi/2 || x == 3*math.Pi/2 || x <= -1 {
		return 0 // Avoid singularity and log domain error
	}
	return math.Exp(math.Sin(x)+math.Cos(x*x)+math.Tan(x)) * math.Log1p(x*x+1)
}

func HyperComplexNonLinear(x float64) float64 {
	if x <= 0 {
		return 0 // Avoid log domain error
	}
	return math.Pow(math.Exp(math.Sin(x)+math.Cos(x)), 1.5) * math.Log1p(x*x*x+1)
}

func UltraComplexNonLinear(x float64) float64 {
	if x == math.Pi/2 || x == 3*math.Pi/2 || x <= -1 || x == 0 {
		return 0 // Avoid singularity and log domain error
	}
	return math.Exp(math.Sin(x*x)+math.Cos(x*x*x)+math.Tan(x)) * math.Log1p(math.Abs(math.Tan(x*x+x)))
}

func MegaComplexNonLinear(x float64) float64 {
	if x == math.Pi/2 || x == 3*math.Pi/2 || x <= -1 {
		return 0 // Avoid singularity and log domain error
	}
	return math.Exp(math.Pow(math.Sin(x), 3)+math.Cos(math.Exp(x))) * math.Log1p(math.Pow(math.Tan(x), 2)+x*x)
}

func GigaComplexNonLinear(x float64) float64 {
	if x <= 0 {
		return 0 // Avoid log domain error
	}
	return math.Exp(math.Sin(x*x)+math.Cos(math.Exp(x))) * math.Log1p(math.Pow(x, 5)+1)
}

func TeraComplexNonLinear(x float64) float64 {
	if x <= -1 {
		return 0 // Avoid log domain error
	}
	return math.Exp(math.Sin(math.Exp(x))+math.Cos(math.Exp(x*x))) * math.Log1p(math.Pow(math.Abs(x), 3)+1)
}

func PetaComplexNonLinear(x float64) float64 {
	if x == math.Pi/2 || x == 3*math.Pi/2 || x <= -1 {
		return 0 // Avoid singularity and log domain error
	}
	return math.Exp(math.Sin(math.Exp(x))+math.Cos(math.Exp(x*x))+math.Tan(math.Exp(x))) * math.Log1p(math.Pow(math.Abs(x), 5)+1)
}

func ExaComplexNonLinear(x float64) float64 {
	if x == math.Pi/2 || x == 3*math.Pi/2 || x <= -1 {
		return 0 // Avoid singularity and log domain error
	}
	return math.Exp(math.Sin(math.Pow(x, 4))+math.Cos(math.Pow(x, 3))+math.Tan(math.Pow(x, 2))) * math.Log1p(math.Exp(math.Abs(x*x+x)))
}

func SuperComplexNonLinear(x float64) float64 {
	if x == math.Pi/2 || x == 3*math.Pi/2 || x <= -1 {
		return 0 // Avoid singularity and log domain error
	}
	return math.Exp(math.Sin(x)*math.Cos(x)+math.Tan(x*x)) * math.Log1p(x*x*x+1)
}

func ExtremlyComplexNonLinear(x float64) float64 {
	if x == math.Pi/2 || x == 3*math.Pi/2 {
		return 0 // Avoid singularity
	}
	return math.Exp(x*x*x) * math.Log1p(math.Abs(math.Tan(x)))
}

func billionFlops(x float64) float64 {
	// Sum inside the exponential function
	sum := float64(0.0)
	for j := 1; j <= 100; j++ {
		sum += math.Pow(x, float64(2*j)) + 1.0/float64(j)
	}

	// Exponential and Logarithm
	expPart := math.Exp(sum)
	logPart := math.Log1p(expPart)

	// Product of trigonometric functions
	product := float64(1.0)
	for i := 1; i <= 1000; i++ {
		powX := math.Pow(x, float64(i))
		product *= math.Sin(powX) + math.Cos(math.Pow(x, float64(i+1))) + math.Tan(powX)
	}

	// Final result
	result := product * logPart
	return result
}

func ComplexNonLinear(x float64) float64 {
	transformFactor := math.Mod(x, 4.0) / 4.0
	if x < 1 {
		if transformFactor < 0.25 {
			return MediumComplexNonLinear(x + (1 + transformFactor))
		} else if transformFactor < 0.5 {
			return MediumComplexNonLinear(x - (1 + transformFactor))
		} else if transformFactor < 0.75 {
			return MediumComplexNonLinear(x * (1 + transformFactor))
		} else {
			return MediumComplexNonLinear(x / (1 + transformFactor))
		}
	} else if x < 10 {
		if transformFactor < 0.25 {
			return IntermediateComplexNonLinear(x + (1 + transformFactor))
		} else if transformFactor < 0.5 {
			return IntermediateComplexNonLinear(x - (1 + transformFactor))
		} else if transformFactor < 0.75 {
			return IntermediateComplexNonLinear(x * (1 + transformFactor))
		} else {
			return IntermediateComplexNonLinear(x / (1 + transformFactor))
		}
	} else {
		if transformFactor < 0.25 {
			return HighComplexNonLinear(x + (1 + transformFactor))
		} else if transformFactor < 0.5 {
			return HighComplexNonLinear(x - (1 + transformFactor))
		} else if transformFactor < 0.75 {
			return HighComplexNonLinear(x * (1 + transformFactor))
		} else {
			return HighComplexNonLinear(x / (1 + transformFactor))
		}
	}
}

func (mat *matrix) computeHoohashRank() int {
	var B [64][64]float64
	for i := range B {
		for j := range B[0] {
			// fmt.Printf("%v\n", mat[i][j])
			B[i][j] = float64(mat[i][j]) + ComplexNonLinear(float64(mat[i][j]))
		}
	}
	var rank int
	var rowSelected [64]bool
	for i := 0; i < 64; i++ {
		var j int
		for j = 0; j < 64; j++ {
			if !rowSelected[j] && math.Abs(B[j][i]) > eps {
				break
			}
		}
		if j != 64 {
			rank++
			rowSelected[j] = true
			for p := i + 1; p < 64; p++ {
				B[j][p] /= B[j][i]
			}
			for k := 0; k < 64; k++ {
				if k != j && math.Abs(B[k][i]) > eps {
					for p := i + 1; p < 64; p++ {
						B[k][p] -= B[j][p] * B[k][i]
					}
				}
			}
		}
	}
	return rank
}

func (mat *matrix) computeRank() int {
	var B [64][64]float64
	for i := range B {
		for j := range B[0] {
			B[i][j] = float64(mat[i][j])
		}
	}
	var rank int
	var rowSelected [64]bool
	for i := 0; i < 64; i++ {
		var j int
		for j = 0; j < 64; j++ {
			if !rowSelected[j] && math.Abs(B[j][i]) > eps {
				break
			}
		}
		if j != 64 {
			rank++
			rowSelected[j] = true
			for p := i + 1; p < 64; p++ {
				B[j][p] /= B[j][i]
			}
			for k := 0; k < 64; k++ {
				if k != j && math.Abs(B[k][i]) > eps {
					for p := i + 1; p < 64; p++ {
						B[k][p] -= B[j][p] * B[k][i]
					}
				}
			}
		}
	}
	return rank
}

func (mat *matrix) HoohashMatrixMultiplicationV1(hash *externalapi.DomainHash) *externalapi.DomainHash {
	hashBytes := hash.ByteArray()
	var vector [64]float64
	var product [64]float64

	// Populate the vector with floating-point values
	for i := 0; i < 32; i++ {
		vector[2*i] = float64(hashBytes[i] >> 4)
		vector[2*i+1] = float64(hashBytes[i] & 0x0F)
	}

	// Matrix-vector multiplication with floating point operations
	for i := 0; i < 64; i++ {
		for j := 0; j < 64; j++ {
			// Transform Matrix values with complex non linear equations and sum into product.
			forComplex := float64(mat[i][j]) * vector[j]
			for forComplex > 16 {
				forComplex = forComplex * 0.1
			}
			product[i] += ComplexNonLinear(forComplex)
		}
	}

	// Convert product back to uint16 and then to byte array
	var res [32]byte
	for i := range res {
		high := uint32(product[2*i] * 0.00000001)
		low := uint32(product[2*i+1] * 0.00000001)
		// Combine high and low into a single byte
		combined := (high ^ low) & 0xFF
		res[i] = hashBytes[i] ^ byte(combined)
	}
	// Hash again
	writer := hashes.Blake3HashWriter()
	writer.InfallibleWrite(res[:])
	return writer.Finalize()
}

func (mat *matrix) HoohashMatrixMultiplicationV101(hash *externalapi.DomainHash) *externalapi.DomainHash {
	hashBytes := hash.ByteArray()
	var vector [64]float64
	var product [64]float64

	// Populate the vector with floating-point values
	for i := 0; i < 32; i++ {
		vector[2*i] = float64(hashBytes[i] >> 4)
		vector[2*i+1] = float64(hashBytes[i] & 0x0F)
	}

	// Matrix-vector multiplication with floating point operations
	for i := 0; i < 64; i++ {
		for j := 0; j < 64; j++ {
			// Transform Matrix values with complex non linear equations and sum into product.
			forComplex := float64(mat[i][j]) * vector[j]
			for forComplex > 14 {
				forComplex = forComplex * 0.1
			}
			product[i] += ComplexNonLinear(forComplex)
		}
	}

	// Convert product back to uint16 and then to byte array
	var res [32]byte
	for i := range res {
		high := uint32(product[2*i] * 0.00000001)
		low := uint32(product[2*i+1] * 0.00000001)
		// Combine high and low into a single byte
		combined := (high ^ low) & 0xFF
		res[i] = hashBytes[i] ^ byte(combined)
	}
	// Hash again
	writer := hashes.Blake3HashWriter()
	writer.InfallibleWrite(res[:])
	return writer.Finalize()
}

const COMPLEX_OUTPUT_CLAMP = 100000
const PRODUCT_VALUE_SCALE_MULTIPLIER = 0.00001

func ForComplex(forComplex float64) float64 {
	var complex float64
	complex = ComplexNonLinear(forComplex)
	for complex >= COMPLEX_OUTPUT_CLAMP {
		forComplex *= 0.1
		complex = ComplexNonLinear(forComplex)
	}
	return complex
}

func (mat *floatMatrix) HoohashMatrixMultiplicationV110(hash *externalapi.DomainHash) *externalapi.DomainHash {
	hashBytes := hash.ByteArray()
	var vector [64]float64
	var product [64]float64

	// Populate the vector with floating-point values from the hash bytes
	for i := 0; i < 32; i++ {
		vector[2*i] = float64(hashBytes[i] >> 4)     // Upper 4 bits
		vector[2*i+1] = float64(hashBytes[i] & 0x0F) // Lower 4 bits
	}

	// Perform the matrix-vector multiplication with nonlinear adjustments
	for i := 0; i < 64; i++ {
		for j := 0; j < 64; j++ {
			switch ((i * j) + int(mat[i][j])*int(vector[j])) % 100 {
			case 0: // Complex non-linear function
				product[i] += ForComplex(mat[i][j] * vector[j])
			case 1, 5, 9, 13, 17: // Division
				if vector[j] != 0 {
					product[i] += mat[i][j] / vector[j]
				} else {
					product[i] += mat[i][j] / 1.0 // Safeguard against division by zero
				}
			case 2, 6, 10, 14, 18: // Multiplication
				product[i] += mat[i][j] * vector[j] * PRODUCT_VALUE_SCALE_MULTIPLIER
			case 3, 7, 11, 15, 19: // Addition
				product[i] += mat[i][j] + vector[j]
			case 4, 8, 12, 16: // Subtraction
				product[i] += mat[i][j] - vector[j]
			default: // Fallback multiplication
				product[i] += mat[i][j] * vector[j] * PRODUCT_VALUE_SCALE_MULTIPLIER
			}
		}
	}

	// Generate the result bytes
	var res [32]uint8
	var scaledValues [32]uint8
	for i := 0; i < 64; i += 2 {
		scaledValues[i/2] = uint8((product[i] + product[i+1]) * PRODUCT_VALUE_SCALE_MULTIPLIER)
	}
	for i := 0; i < 32; i++ {
		res[i] = hashBytes[i] ^ scaledValues[i]
	}
	writer := hashes.Blake3HashWriter()
	writer.InfallibleWrite(res[:32])
	return writer.Finalize()
}

func (mat *matrix) bHeavyHash(hash *externalapi.DomainHash) *externalapi.DomainHash {
	hashBytes := hash.ByteArray()
	var vector [64]uint16
	var product [64]uint16
	for i := 0; i < 32; i++ {
		vector[2*i] = uint16(hashBytes[i] >> 4)
		vector[2*i+1] = uint16(hashBytes[i] & 0x0F)
	}
	// Matrix-vector multiplication, and convert to 4 bits.
	for i := 0; i < 64; i++ {
		var sum uint16
		for j := 0; j < 64; j++ {
			sum += mat[i][j] * vector[j]
		}
		product[i] = sum >> 10
	}

	// Concatenate 4 LSBs back to 8 bit xor with sum1
	var res [32]byte
	for i := range res {
		res[i] = hashBytes[i] ^ (byte(product[2*i]<<4) | byte(product[2*i+1]))
	}
	// Hash again
	writer := hashes.BlakeHeavyHashWriter()
	writer.InfallibleWrite(res[:])
	return writer.Finalize()
}

func (mat *matrix) hHeavyHash(hash *externalapi.DomainHash) *externalapi.DomainHash {
	hashBytes := hash.ByteArray()
	var vector [64]uint16
	var product [64]uint16
	for i := 0; i < 32; i++ {
		vector[2*i] = uint16(hashBytes[i] >> 4)
		vector[2*i+1] = uint16(hashBytes[i] & 0x0F)
	}
	// Matrix-vector multiplication, and convert to 4 bits.
	for i := 0; i < 64; i++ {
		var sum uint16
		for j := 0; j < 64; j++ {
			sum += mat[i][j] * vector[j]
		}
		product[i] = sum >> 10
	}

	// Concatenate 4 LSBs back to 8 bit xor with sum1
	var res [32]byte
	for i := range res {
		res[i] = hashBytes[i] ^ (byte(product[2*i]<<4) | byte(product[2*i+1]))
	}
	// Hash again
	writer := hashes.BlakeHeavyHashWriter()
	writer.InfallibleWrite(res[:])
	return writer.Finalize()
}

func (mat *matrix) kHeavyHash(hash *externalapi.DomainHash) *externalapi.DomainHash {
	hashBytes := hash.ByteArray()
	var vector [64]uint16
	var product [64]uint16
	for i := 0; i < 32; i++ {
		vector[2*i] = uint16(hashBytes[i] >> 4)
		vector[2*i+1] = uint16(hashBytes[i] & 0x0F)
	}
	// Matrix-vector multiplication, and convert to 4 bits.
	for i := 0; i < 64; i++ {
		var sum uint16
		for j := 0; j < 64; j++ {
			sum += mat[i][j] * vector[j]
		}
		product[i] = sum >> 10
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
