package game

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"

	"aviator-backend/utils"
)

// GenerateCrashPoint generates a provably fair crash point.

//

// Model:

//  1. HMAC-SHA256(serverSeed, clientSeed:nonce)

//  2. About 1% of rounds instant-crash at 1.00x via divisibility pre-check

//  3. Remaining rounds use raw = 2^32 / (h+1)

//  4. Minimum non-instant result is 1.01x

//  5. Result is truncated to 2 decimals

//

// Notes:

// - This implementation is house-profitable, but not "exact 1% at every cashout target".

// - 1.01x is intentionally close to break-even because only the 1.00x bucket loses there.

// - Decimal truncation adds a small extra house edge.
func GenerateCrashPoint(serverSeed, clientSeed string, nonce int64) float64 {
	message := fmt.Sprintf("%s:%d", clientSeed, nonce)
	mac := hmac.New(sha256.New, []byte(serverSeed))
	mac.Write([]byte(message))
	hashBytes := mac.Sum(nil)

	// House edge lives here — 1 in 100 rounds crash instantly
	const instantCrashDivisor = 100
	if isDivisible(hashBytes, instantCrashDivisor) {
		return 1.00
	}

	h := binary.BigEndian.Uint32(hashBytes[:4])

	const maxUint32 = float64(1 << 32)

	// Primary house edge comes from the instant-crash pre-check.
	// Decimal truncation adds a small additional edge.
	raw := maxUint32 / float64(h+1)

	// Formula path must never produce 1.00x (isDivisible owns that bucket)
	crashPoint := math.Trunc(raw*100+1e-9) / 100

	if crashPoint < 1.01 {

		return 1.01

	}
	return math.Min(crashPoint, 1_000_000)
}

// isDivisible checks whether the hash (interpreted as a big-endian integer)
// is divisible by divisor, computed incrementally to avoid big.Int overhead.
func isDivisible(hash []byte, divisor int) bool {
	if divisor <= 0 {
		return false
	}
	remainder := 0
	for _, b := range hash {
		remainder = ((remainder << 8) + int(b)) % divisor
	}
	return remainder == 0
}

// GenerateHash returns the HMAC-SHA256 hash for a given round.
// Players use this to independently verify crash points.
func GenerateHash(serverSeed, clientSeed string, nonce int64) string {
	message := fmt.Sprintf("%s:%d", clientSeed, nonce)
	mac := hmac.New(sha256.New, []byte(serverSeed))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyRound verifies all provably fair steps for a completed round.
// Returns true only if seed hash, HMAC, and crash point all match.
func VerifyRound(serverSeed, serverSeedHash, clientSeed string, nonce int64, hash string, crashPoint float64) (bool, map[string]interface{}) {
	steps := make(map[string]interface{})

	// Step 1: Verify server seed commitment
	calculatedSeedHash := utils.SHA256Hash(serverSeed)
	seedHashMatch := calculatedSeedHash == serverSeedHash
	steps["seed_hash_match"] = seedHashMatch
	steps["calculated_seed_hash"] = calculatedSeedHash
	steps["provided_seed_hash"] = serverSeedHash

	// Step 2: Verify HMAC hash
	calculatedHash := GenerateHash(serverSeed, clientSeed, nonce)
	hmacMatch := calculatedHash == hash
	steps["hmac_match"] = hmacMatch
	steps["calculated_hash"] = calculatedHash
	steps["provided_hash"] = hash

	// Step 3: Verify crash point
	calculatedCrashPoint := GenerateCrashPoint(serverSeed, clientSeed, nonce)
	crashPointMatch := calculatedCrashPoint == crashPoint
	steps["crash_point_match"] = crashPointMatch
	steps["calculated_crash_point"] = calculatedCrashPoint
	steps["provided_crash_point"] = crashPoint

	valid := seedHashMatch && hmacMatch && crashPointMatch
	steps["valid"] = valid

	return valid, steps
}

// IsInstantCrash reports whether a crash point is the minimum (1.00x).
func IsInstantCrash(crashPoint float64) bool {
	return crashPoint <= 1.00
}
