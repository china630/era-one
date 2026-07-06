// Package shamir — threshold secret sharing (GF(251), ADR-0013 §2).
package shamir

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
)

const prime = 251

// Split делит secret на parts долей; threshold — минимум для восстановления.
// Секрет кодируется в hex, чтобы все коэффициенты < 251 (GF(251)).
func Split(secret []byte, parts, threshold int) ([][]byte, error) {
	if len(secret) == 0 {
		return nil, errors.New("empty secret")
	}
	return splitBytes([]byte(hex.EncodeToString(secret)), parts, threshold)
}

// Combine восстанавливает secret из >= threshold долей.
func Combine(shares [][]byte) ([]byte, error) {
	raw, err := combineBytes(shares)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(string(raw))
}

func splitBytes(secret []byte, parts, threshold int) ([][]byte, error) {
	if parts < threshold || threshold < 2 {
		return nil, errors.New("invalid threshold")
	}
	shares := make([][]byte, parts)
	for i := range shares {
		shares[i] = make([]byte, len(secret)+1)
		shares[i][0] = byte(i + 1)
	}
	for idx, b := range secret {
		coeffs := make([]int, threshold)
		coeffs[0] = int(b)
		buf := make([]byte, threshold-1)
		if _, err := rand.Read(buf); err != nil {
			return nil, err
		}
		for j := 1; j < threshold; j++ {
			coeffs[j] = int(buf[j-1]) % prime
		}
		for i := 0; i < parts; i++ {
			x := int(shares[i][0])
			shares[i][idx+1] = byte(evalPoly(coeffs, x))
		}
	}
	return shares, nil
}

func combineBytes(shares [][]byte) ([]byte, error) {
	if len(shares) < 2 {
		return nil, errors.New("need shares")
	}
	secretLen := len(shares[0]) - 1
	if secretLen <= 0 {
		return nil, errors.New("bad share")
	}
	secret := make([]byte, secretLen)
	for idx := 0; idx < secretLen; idx++ {
		points := make([][2]int, 0, len(shares))
		for _, sh := range shares {
			if len(sh) != secretLen+1 {
				return nil, errors.New("share length mismatch")
			}
			points = append(points, [2]int{int(sh[0]), int(sh[idx+1])})
		}
		secret[idx] = byte(lagrange(points, 0))
	}
	return secret, nil
}
func EncodeShares(shares [][]byte) []string {
	out := make([]string, len(shares))
	for i, s := range shares {
		out[i] = hex.EncodeToString(s)
	}
	return out
}

// DecodeShares из hex.
func DecodeShares(encoded []string) ([][]byte, error) {
	out := make([][]byte, len(encoded))
	for i, s := range encoded {
		b, err := hex.DecodeString(s)
		if err != nil {
			return nil, err
		}
		out[i] = b
	}
	return out, nil
}

func evalPoly(coeffs []int, x int) int {
	result := 0
	xPow := 1
	for _, c := range coeffs {
		result = (result + c*xPow) % prime
		xPow = (xPow * x) % prime
	}
	return result
}

func lagrange(points [][2]int, at int) int {
	result := 0
	for i, pi := range points {
		num := 1
		den := 1
		for j, pj := range points {
			if i == j {
				continue
			}
			num = (num * (at - pj[0])) % prime
			den = (den * (pi[0] - pj[0])) % prime
		}
		inv := modInverse(den, prime)
		term := (pi[1] * num % prime) * inv % prime
		result = (result + term) % prime
	}
	if result < 0 {
		result += prime
	}
	return result
}

func modInverse(a, m int) int {
	a = a % m
	if a < 0 {
		a += m
	}
	for x := 1; x < m; x++ {
		if (a*x)%m == 1 {
			return x
		}
	}
	return 1
}
