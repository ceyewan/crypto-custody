package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strings"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func main() {
	var resultFile string
	var messageHash string
	var signature string
	var address string
	flag.StringVar(&resultFile, "result", "", "offline sign result JSON file")
	flag.StringVar(&messageHash, "hash", "", "32-byte message hash hex")
	flag.StringVar(&signature, "signature", "", "65-byte Ethereum r||s||v signature hex")
	flag.StringVar(&address, "address", "", "expected Ethereum address")
	flag.Parse()

	if resultFile != "" {
		if err := loadResult(resultFile, &messageHash, &signature, &address); err != nil {
			fail(err)
		}
	}

	report, err := verifyEthereumSignature(messageHash, signature, address)
	if err != nil {
		printReport(report)
		fail(err)
	}
	report["success"] = true
	printReport(report)
}

func loadResult(path string, messageHash, signature, address *string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var pkg struct {
		Payload struct {
			MessageHash string `json:"message_hash"`
			Signature   string `json:"signature"`
			FromAddress string `json:"from_address"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return err
	}
	if *messageHash == "" {
		*messageHash = pkg.Payload.MessageHash
	}
	if *signature == "" {
		*signature = pkg.Payload.Signature
	}
	if *address == "" {
		*address = pkg.Payload.FromAddress
	}
	return nil
}

func verifyEthereumSignature(messageHash, signature, expectedAddress string) (map[string]any, error) {
	report := map[string]any{
		"message_hash":     messageHash,
		"signature":        signature,
		"expected_address": expectedAddress,
		"signature_format": "ethereum_rsv",
	}
	hashBytes, err := decodeFixedHex(messageHash, 32, "message_hash")
	if err != nil {
		return report, err
	}
	signatureBytes, err := decodeFixedHex(signature, 65, "signature")
	if err != nil {
		return report, err
	}
	if signatureBytes[64] == 27 || signatureBytes[64] == 28 {
		signatureBytes[64] -= 27
	}
	if signatureBytes[64] != 0 && signatureBytes[64] != 1 {
		return report, fmt.Errorf("signature recovery id invalid: %d", signatureBytes[64])
	}

	r := new(big.Int).SetBytes(signatureBytes[:32])
	s := new(big.Int).SetBytes(signatureBytes[32:64])
	report["low_s_valid"] = ethcrypto.ValidateSignatureValues(signatureBytes[64], r, s, true)

	var attempts []map[string]any
	for _, hashVariant := range hashVariants(messageHash, hashBytes) {
		for _, signatureVariant := range signatureVariants(signatureBytes) {
			for _, recoveryID := range []byte{signatureVariant.bytes[64], 1 - signatureVariant.bytes[64]} {
				candidate := append([]byte(nil), signatureVariant.bytes...)
				candidate[64] = recoveryID
				attempt := map[string]any{
					"hash_variant":      hashVariant.name,
					"signature_variant": signatureVariant.name,
					"recovery_id":       int(recoveryID),
				}
				pubKey, err := ethcrypto.SigToPub(hashVariant.bytes, candidate)
				if err != nil {
					attempt["error"] = err.Error()
					attempts = append(attempts, attempt)
					continue
				}
				recovered := ethcrypto.PubkeyToAddress(*pubKey).Hex()
				attempt["address"] = recovered
				attempts = append(attempts, attempt)
				if hashVariant.name == "direct" && signatureVariant.name == "normal" && strings.EqualFold(recovered, expectedAddress) {
					report["recovered_address"] = recovered
					report["recovery_id"] = int(recoveryID)
					report["hash_variant"] = hashVariant.name
					report["signature_variant"] = signatureVariant.name
					report["attempts"] = attempts
					return report, nil
				}
			}
		}
	}
	report["attempts"] = attempts
	return report, fmt.Errorf("signature does not recover expected address")
}

type hashVariant struct {
	name  string
	bytes []byte
}

func hashVariants(messageHash string, hashBytes []byte) []hashVariant {
	hexText := []byte(strings.TrimPrefix(strings.TrimSpace(messageHash), "0x"))
	shaRaw := sha256.Sum256(hashBytes)
	shaHex := sha256.Sum256(hexText)
	asciiHexModN := bigintModNBytes(hexText)
	return []hashVariant{
		{name: "direct", bytes: hashBytes},
		{name: "ascii_hex_bigint_mod_n", bytes: asciiHexModN},
		{name: "sha256_raw_hash", bytes: shaRaw[:]},
		{name: "sha256_hex_text", bytes: shaHex[:]},
		{name: "keccak_raw_hash", bytes: ethcrypto.Keccak256(hashBytes)},
		{name: "keccak_hex_text", bytes: ethcrypto.Keccak256(hexText)},
	}
}

func bigintModNBytes(value []byte) []byte {
	scalar := new(big.Int).SetBytes(value)
	scalar.Mod(scalar, ethcrypto.S256().Params().N)
	return leftPadTo32Bytes(scalar.Bytes())
}

type signatureVariant struct {
	name  string
	bytes []byte
}

func signatureVariants(signature []byte) []signatureVariant {
	normal := append([]byte(nil), signature...)
	reverseR := append([]byte(nil), signature...)
	reverseBytes(reverseR[:32])
	reverseS := append([]byte(nil), signature...)
	reverseBytes(reverseS[32:64])
	reverseRS := append([]byte(nil), signature...)
	reverseBytes(reverseRS[:32])
	reverseBytes(reverseRS[32:64])
	return []signatureVariant{
		{name: "normal", bytes: normal},
		{name: "reverse_r", bytes: reverseR},
		{name: "reverse_s", bytes: reverseS},
		{name: "reverse_r_s", bytes: reverseRS},
	}
}

func reverseBytes(value []byte) {
	for i, j := 0, len(value)-1; i < j; i, j = i+1, j-1 {
		value[i], value[j] = value[j], value[i]
	}
}

func leftPadTo32Bytes(value []byte) []byte {
	if len(value) >= 32 {
		return value[len(value)-32:]
	}
	padded := make([]byte, 32)
	copy(padded[32-len(value):], value)
	return padded
}

func decodeFixedHex(value string, length int, label string) ([]byte, error) {
	cleaned := strings.TrimPrefix(strings.TrimSpace(value), "0x")
	decoded, err := hex.DecodeString(cleaned)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", label, err)
	}
	if len(decoded) != length {
		return nil, fmt.Errorf("%s length=%d, want %d bytes", label, len(decoded), length)
	}
	return decoded, nil
}

func printReport(report map[string]any) {
	raw, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(raw))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "FAIL:", err)
	os.Exit(1)
}
