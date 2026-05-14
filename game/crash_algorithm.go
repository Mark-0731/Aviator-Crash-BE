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

// GenerateCrashPoint generates a provably fair crash point
// Uses the same algorithm as Stake.com and BC.game (2^32 method)
//
// Algorithm:
//  1. HMAC-SHA256(serverSeed, clientSeed) → 32-byte hash
//  2. Read first 4 bytes as a big-endian uint32
//  3. Apply house-edge formula: (2^32 / (uint32 + 1)) * (1 - houseEdge)
//  4. Floor to 2 decimal places, minimum 1.00x
//
// This is cryptographically identical to Stake's Crash implementation.
func GenerateCrashPoint(serverSeed, clientSeed string, nonce int64) float64 {
	// Step 1: HMAC-SHA256(serverSeed, clientSeed:nonce)
	// The nonce is appended to the client seed so each round is unique
	message := fmt.Sprintf("%s:%d", clientSeed, nonce)
	mac := hmac.New(sha256.New, []byte(serverSeed))
	mac.Write([]byte(message))
	hashBytes := mac.Sum(nil)

	// Step 2: Read the first 4 bytes as a big-endian uint32
	uint32Value := binary.BigEndian.Uint32(hashBytes[:4])

	// Step 3: Apply Stake/BC.game formula
	// houseEdge = 0.01 (1%)
	// result = (2^32 / (uint32 + 1)) * (1 - houseEdge)
	const houseEdge = 0.01
	const maxUint32 = float64(1 << 32) // 4294967296
	rawCrashPoint := (maxUint32 / float64(uint32Value+1)) * (1 - houseEdge)

	// Step 4: Clamp between 1.00 and 1,000,000x, floor to 2 decimal places
	// No artificial 20x cap — large multipliers can happen just like Stake
	crashPoint := math.Max(1.00, math.Min(rawCrashPoint, 1_000_000))
	crashPoint = math.Floor(crashPoint*100) / 100

	return crashPoint
}

// GenerateHash generates the HMAC-SHA256 hash for verification
// Message format matches Stake/BC.game: HMAC-SHA256(serverSeed, clientSeed:nonce)
func GenerateHash(serverSeed, clientSeed string, nonce int64) string {
	message := fmt.Sprintf("%s:%d", clientSeed, nonce)
	mac := hmac.New(sha256.New, []byte(serverSeed))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyRound verifies the provably fair calculation
// Returns true if all verification steps pass
func VerifyRound(serverSeed, serverSeedHash, clientSeed string, nonce int64, hash string, crashPoint float64) (bool, map[string]interface{}) {
	steps := make(map[string]interface{})

	// Step 1: Verify server seed hash (commitment)
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

	// Step 3: Verify crash point calculation
	// Both values are floored to 2 decimals, so exact equality works
	calculatedCrashPoint := GenerateCrashPoint(serverSeed, clientSeed, nonce)
	crashPointMatch := calculatedCrashPoint == crashPoint
	steps["crash_point_match"] = crashPointMatch
	steps["calculated_crash_point"] = calculatedCrashPoint
	steps["provided_crash_point"] = crashPoint

	// All steps must pass
	valid := seedHashMatch && hmacMatch && crashPointMatch
	steps["valid"] = valid

	return valid, steps
}

// IsInstantCrash checks if a crash point is at 1.00x
func IsInstantCrash(crashPoint float64) bool {
	return crashPoint <= 1.00
}
