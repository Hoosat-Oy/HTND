package pow

import (
	"math"

	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/hashes"
)

const eps float64 = 1e-9

// type matrix [64][64]uint16
type matrix [64][64]uint16

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

func generateMatrix(hash *externalapi.DomainHash) *matrix {
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

func generateHoohashMatrix(hash *externalapi.DomainHash) *matrix {
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
    return math.Exp(math.Sin(x) + math.Cos(x*x)) * math.Log1p(x*x + 1)
}


func HighComplexNonLinear(x float64) float64 {
	return math.Exp(x) * math.Log(x + 1)
}

func VeryComplexNonLinear(x float64) float64 {
    if x == math.Pi/2 || x == 3*math.Pi/2 || x <= -1 {
        return 0 // Avoid singularity and log domain error
    }
    return math.Exp(math.Sin(x) + math.Cos(x*x) + math.Tan(x)) * math.Log1p(x*x + 1)
}

func HyperComplexNonLinear(x float64) float64 {
    if x <= 0 {
        return 0 // Avoid log domain error
    }
    return math.Pow(math.Exp(math.Sin(x) + math.Cos(x)), 1.5) * math.Log1p(x*x*x + 1)
}

func UltraComplexNonLinear(x float64) float64 {
    if x == math.Pi/2 || x == 3*math.Pi/2 || x <= -1 || x == 0 {
        return 0 // Avoid singularity and log domain error
    }
    return math.Exp(math.Sin(x*x) + math.Cos(x*x*x) + math.Tan(x)) * math.Log1p(math.Abs(math.Tan(x*x + x)))
}

func MegaComplexNonLinear(x float64) float64 {
    if x == math.Pi/2 || x == 3*math.Pi/2 || x <= -1 {
        return 0 // Avoid singularity and log domain error
    }
    return math.Exp(math.Pow(math.Sin(x), 3) + math.Cos(math.Exp(x))) * math.Log1p(math.Pow(math.Tan(x), 2) + x*x)
}

func GigaComplexNonLinear(x float64) float64 {
    if x <= 0 {
        return 0 // Avoid log domain error
    }
    return math.Exp(math.Sin(x*x) + math.Cos(math.Exp(x))) * math.Log1p(math.Pow(x, 5) + 1)
}

func TeraComplexNonLinear(x float64) float64 {
    if x <= -1 {
        return 0 // Avoid log domain error
    }
    return math.Exp(math.Sin(math.Exp(x)) + math.Cos(math.Exp(x*x))) * math.Log1p(math.Pow(math.Abs(x), 3) + 1)
}

func PetaComplexNonLinear(x float64) float64 {
    if x == math.Pi/2 || x == 3*math.Pi/2 || x <= -1 {
        return 0 // Avoid singularity and log domain error
    }
    return math.Exp(math.Sin(math.Exp(x)) + math.Cos(math.Exp(x*x)) + math.Tan(math.Exp(x))) * math.Log1p(math.Pow(math.Abs(x), 5) + 1)
}

func ExaComplexNonLinear(x float64) float64 {
    if x == math.Pi/2 || x == 3*math.Pi/2 || x <= -1 {
        return 0 // Avoid singularity and log domain error
    }
    return math.Exp(math.Sin(math.Pow(x, 4)) + math.Cos(math.Pow(x, 3)) + math.Tan(math.Pow(x, 2))) * math.Log1p(math.Exp(math.Abs(x*x + x)))
}

func SuperComplexNonLinear(x float64) float64 {
    if x == math.Pi/2 || x == 3*math.Pi/2 || x <= -1 {
        return 0 // Avoid singularity and log domain error
    }
    return math.Exp(math.Sin(x)*math.Cos(x) + math.Tan(x*x)) * math.Log1p(x*x*x + 1)
}

func ExtremlyComplexNonLinear(x float64) float64 {
    if x == math.Pi/2 || x == 3*math.Pi/2 {
        return 0 // Avoid singularity
    }
    return math.Exp(x*x*x) * math.Log1p(math.Abs(math.Tan(x)))
}

func ComplexNonLinear(x float64) float64 {
	transformFactor := math.Mod(x, 1.0)
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


func (mat *matrix) HoohashMatrixMultiplication(hash *externalapi.DomainHash) []byte {
	hashBytes := hash.ByteArray()
	var vector [64]float64
	var product [64]float64
	for i := 0; i < 32; i++ {
		vector[2*i] = float64(hashBytes[i] >> 4)
		vector[2*i+1] = float64(hashBytes[i] & 0x0F)
	}
	// Matrix-vector multiplication with floating point operations
	for i := 0; i < 64; i++ {
		var sum float64
		for j := 0; j < 64; j++ {
			sum += float64(mat[i][j]) * ComplexNonLinear(vector[j]) // Introduce non-linear operations
		}
		product[i] = sum
	}

	// Convert product back to uint16 and then to byte array
	var res [32]byte
	for i := range res {
		high := uint16(math.Mod(product[2*i], 16))
		low := uint16(math.Mod(product[2*i+1], 16))
		res[i] = hashBytes[i] ^ (byte(high<<4) | byte(low))
	}
	// Hash again
	writer := hashes.BlakeHeavyHashWriter()
	writer.InfallibleWrite(res[:])
	return res[:]
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


func (mat *matrix) walahash(hash *externalapi.DomainHash) *externalapi.DomainHash {
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
	keccakWriter := hashes.KeccakHeavyHashWriter()
	keccakWriter.InfallibleWrite(res[:])
	blakeWriter := hashes.BlakeHeavyHashWriter()
	keccakFinalized := keccakWriter.Finalize()
	blakeWriter.InfallibleWrite([]byte(keccakFinalized.String()))
	return blakeWriter.Finalize()
}


func (mat *matrix) ComplexNonLinearkHeavyHash(hash *externalapi.DomainHash) *externalapi.DomainHash {
	hashBytes := hash.ByteArray()
	var vector [64]float64
	var product [64]float64
	for i := 0; i < 32; i++ {
		vector[2*i] = float64(hashBytes[i] >> 4)
		vector[2*i+1] = float64(hashBytes[i] & 0x0F)
	}
	// Matrix-vector multiplication with floating point operations
	for i := 0; i < 64; i++ {
		var sum float64
		for j := 0; j < 64; j++ {
			sum += float64(mat[i][j]) * BasicComplexNonLinear(vector[j]) // Introduce non-linear operations
		}
		product[i] = sum
	}

	// Convert product back to uint16 and then to byte array
	var res [32]byte
	for i := range res {
		high := uint16(math.Mod(product[2*i], 16))
		low := uint16(math.Mod(product[2*i+1], 16))
		res[i] = hashBytes[i] ^ (byte(high<<4) | byte(low))
	}
	// Hash again
	writer := hashes.KeccakHeavyHashWriter()
	writer.InfallibleWrite(res[:])
	return writer.Finalize()
}