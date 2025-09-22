package hd

import (
	"github.com/cosmos/go-bip39"

	"github.com/cosmos/cosmos-sdk/crypto/keys/dilithium"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/types"
)

// PubKeyType defines an algorithm to derive key-pairs which can be used for cryptographic signing.
type PubKeyType string

const (
	// MultiType implies that a pubkey is a multisignature
	MultiType = PubKeyType("multi")
	// Secp256k1Type uses the Bitcoin secp256k1 ECDSA parameters.
	Secp256k1Type = PubKeyType("secp256k1")
	// Ed25519Type represents the Ed25519Type signature system.
	// It is currently not supported for end-user keys (wallets/ledgers).
	Ed25519Type = PubKeyType("ed25519")
	// Bls12_381Type represents the Bls12_381Type signature system.
	// It is currently not supported for end-user keys (wallets/ledgers).
	Bls12_381Type = PubKeyType("bls12_381")
	// Dilithium2Type represents the Post-Quantum Dilithium2 signature system.
	// Supported for software keys only (no Ledger support).
	Dilithium2Type = PubKeyType("dilithium2")
)

// Secp256k1 uses the Bitcoin secp256k1 ECDSA parameters.
var Secp256k1 = secp256k1Algo{}

// Dilithium2 is a post-quantum key algorithm backed by software implementation.
var Dilithium2 = dilithium2Algo{}

type (
	DeriveFn   func(mnemonic, bip39Passphrase, hdPath string) ([]byte, error)
	GenerateFn func(bz []byte) types.PrivKey
)

type WalletGenerator interface {
	Derive(mnemonic, bip39Passphrase, hdPath string) ([]byte, error)
	Generate(bz []byte) types.PrivKey
}

type secp256k1Algo struct{}

func (s secp256k1Algo) Name() PubKeyType {
	return Secp256k1Type
}

// Derive derives and returns the secp256k1 private key for the given seed and HD path.
func (s secp256k1Algo) Derive() DeriveFn {
	return func(mnemonic, bip39Passphrase, hdPath string) ([]byte, error) {
		seed, err := bip39.NewSeedWithErrorChecking(mnemonic, bip39Passphrase)
		if err != nil {
			return nil, err
		}

		masterPriv, ch := ComputeMastersFromSeed(seed)
		if len(hdPath) == 0 {
			return masterPriv[:], nil
		}
		derivedKey, err := DerivePrivateKeyForPath(masterPriv, ch, hdPath)

		return derivedKey, err
	}
}

// Generate generates a secp256k1 private key from the given bytes.
func (s secp256k1Algo) Generate() GenerateFn {
	return func(bz []byte) types.PrivKey {
		bzArr := make([]byte, secp256k1.PrivKeySize)
		copy(bzArr, bz)

		return &secp256k1.PrivKey{Key: bzArr}
	}
}

// ---------------------- Dilithium2 ----------------------

type dilithium2Algo struct{}

func (d dilithium2Algo) Name() PubKeyType { return Dilithium2Type }

// Derive returns a BIP39-based derivation function. We reuse the same master/child
// derivation used for secp256k1 to obtain deterministic key bytes, which are then
// expanded to the Dilithium private key size in Generate().
func (d dilithium2Algo) Derive() DeriveFn {
	return func(mnemonic, bip39Passphrase, hdPath string) ([]byte, error) {
		seed, err := bip39.NewSeedWithErrorChecking(mnemonic, bip39Passphrase)
		if err != nil {
			return nil, err
		}

		masterPriv, ch := ComputeMastersFromSeed(seed)
		if len(hdPath) == 0 {
			return masterPriv[:], nil
		}
		derivedKey, err := DerivePrivateKeyForPath(masterPriv, ch, hdPath)
		return derivedKey, err
	}
}

// Generate creates a Dilithium private key by expanding the input bytes to
// the expected private key size using a simple deterministic repetition.
func (d dilithium2Algo) Generate() GenerateFn {
	return func(bz []byte) types.PrivKey {
		// Ensure non-empty seed
		if len(bz) == 0 {
			bz = []byte{0x42}
		}
		out := make([]byte, dilithium.PrivKeySize)
		for i := 0; i < len(out); i++ {
			out[i] = bz[i%len(bz)]
		}
		return &dilithium.PrivKey{Key: out}
	}
}
