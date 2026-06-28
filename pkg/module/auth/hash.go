package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 3         // iterations
	argonMemory  = 64 * 1024 // 64 MB in KiB
	argonThreads = 2         // parallelism
	argonKeyLen  = 32        // 32 bytes output
	saltLen      = 16        // 16 bytes salt
)

// HashPassword returns a PHC-style encoded argon2id string to store in DB.
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64Salt, b64Hash,
	)

	return encoded, nil
}

// ComparePasswordAndHash verifies password against the PHC-style encoded hash.
// needsRehash == true if the stored hash uses weaker params than current defaults.
//
//nolint:funlen
func ComparePasswordAndHash(password, encodedHash string) (match bool, needsRehash bool, err error) {
	parts := strings.Split(encodedHash, "$")
	// Expected parts: ["", "argon2id", "v=19", "m=...,t=...,p=...", "salt", "hash"]
	if len(parts) < 6 {
		return false, false, errors.New("invalid hash format")
	}
	if parts[1] != "argon2id" {
		return false, false, errors.New("unsupported algorithm")
	}

	// Version
	vStr := strings.TrimPrefix(parts[2], "v=")
	version, err := strconv.Atoi(vStr)
	if err != nil {
		return false, false, fmt.Errorf("bad version: %w", err)
	}
	if version != argon2.Version {
		return false, false, fmt.Errorf("unsupported algorithm version: %d", version)
	}

	// Params
	var mem uint32
	var time uint32
	var threads uint8
	for p := range strings.SplitSeq(parts[3], ",") {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			return false, false, errors.New("invalid params")
		}
		k, v := kv[0], kv[1]
		switch k {
		case "m":
			var m uint64
			m, err = strconv.ParseUint(v, 10, 32)
			if err != nil {
				return false, false, err
			}
			mem = uint32(m)
		case "t":
			var t uint64
			t, err = strconv.ParseUint(v, 10, 32)
			if err != nil {
				return false, false, err
			}
			time = uint32(t)
		case "p":
			var pv uint64
			pv, err = strconv.ParseUint(v, 10, 8)
			if err != nil {
				return false, false, err
			}
			threads = uint8(pv)
		default:
			// Ignore unknown params
		}
	}

	// Salt & hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, false, fmt.Errorf("bad salt encoding: %w", err)
	}

	actualHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, false, fmt.Errorf("bad password hash encoding: %w", err)
	}
	hashLen := len(actualHash)
	if hashLen >= 4096 {
		return false, false, fmt.Errorf("password hash too long: %d", hashLen)
	}

	// Compute hash with extracted params
	computed := argon2.IDKey([]byte(password), salt, time, mem, threads, uint32(hashLen))

	// Constant-time compare
	if subtle.ConstantTimeCompare(computed, actualHash) == 1 {
		// Check whether params are weaker than current defaults -> suggest rehash
		needsRehash = (mem < argonMemory) || (time < argonTime) || (threads < argonThreads) ||
			(len(actualHash) < argonKeyLen)
		return true, needsRehash, nil
	}

	return false, false, nil
}
