package probably

import (
	"math"
)

const (
	pow_2_32    float64 = 4294967296
	negpow_2_32 float64 = -4294967296
	alpha_16    float64 = 0.673
	alpha_32    float64 = 0.697
	alpha_64    float64 = 0.709
)

// A HyperLogLog cardinality estimator.
//
// See http://algo.inria.fr/flajolet/Publications/FlFuGaMe07.pdf for
// more information.
type HyperLogLog struct {
	m       uint
	k       float64
	k_comp  int
	alpha_m float64
	bits    []uint8
}

// Get a HyperLogLog to count within the given stderr.
//
// Smaller values require more space, but provide more accurate
// results.  For a good time, try 0.001 or so.
func NewHyperLogLog(std_err float64) *HyperLogLog {
	rv := &HyperLogLog{}

	m := 1.04 / std_err
	rv.k = math.Ceil(math.Log2(m * m))
	rv.k_comp = int(32 - rv.k)
	rv.m = uint(math.Pow(2.0, rv.k))

	switch rv.m {
	case 16:
		rv.alpha_m = alpha_16
	case 32:
		rv.alpha_m = alpha_32
	case 64:
		rv.alpha_m = alpha_64
	default:
		rv.alpha_m = 0.7213 / (1 + 1.079/m)
	}

	rv.bits = make([]uint8, rv.m)

	return rv
}

// Add an item by its hash.
func (h *HyperLogLog) Add(hash uint32) {
	r := 1
	for (hash&1) == 0 && r <= h.k_comp {
		r++
		hash >>= 1
	}

	j := hash >> uint(h.k_comp)
	if r > int(h.bits[j]) {
		h.bits[j] = uint8(r)
	}
}

// Get the current estimate of the number of items seen.
func (h *HyperLogLog) Count() uint64 {
	c := 0.0
	for i := uint(0); i < h.m; i++ {
		c += (1 / math.Pow(2.0, float64(h.bits[i])))
	}
	E := h.alpha_m * float64(h.m*h.m) / c

	// -- make corrections

	if E <= 5/2*float64(h.m) {
		V := float64(0)
		for i := uint(0); i < h.m; i++ {
			if h.bits[i] == 0 {
				V++
			}
		}
		if V > 0 {
			E = float64(h.m) * math.Log(float64(h.m)/V)
		}
	} else if E > 1/30*pow_2_32 {
		E = negpow_2_32 * math.Log(1-E/pow_2_32)
	}
	return uint64(E)
}

// Merge another HyperLogLog into this one.
func (h *HyperLogLog) Merge(from *HyperLogLog) {
	if len(h.bits) != len(from.bits) {
		panic("HLLs are incompatible. They must have the same basis")
	}

	for i, v := range from.bits {
		h.bits[i] += v
	}
}
