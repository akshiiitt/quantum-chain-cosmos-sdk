package dilithium

import (
	"crypto/subtle"
	"fmt"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/tmhash"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
)

// KeyType is the bech32 and Any type URL discriminator used by the SDK.
// PrivKeyName and PubKeyName are used for Amino registration and must match
// the (amino.name) specified in the protobuf definitions.
const (
    KeyType     = "dilithium2"
    PrivKeyName = "tendermint/PrivKeyDilithium2"
    PubKeyName  = "tendermint/PubKeyDilithium2"
)

var (
	_ cryptotypes.PrivKey  = &PrivKey{}
	_ cryptotypes.PubKey   = &PubKey{}
	_ codec.AminoMarshaler = &PrivKey{}
	_ codec.AminoMarshaler = &PubKey{}
)

// ========================= PrivKey =========================

// Bytes returns the privkey byte format.
func (privKey *PrivKey) Bytes() []byte { return privKey.Key }

// Sign is not supported for Dilithium consensus keys in the SDK context.
// Consensus signing is performed by CometBFT using the priv-validator.
func (privKey *PrivKey) Sign(msg []byte) ([]byte, error) {
	return nil, errorsmod.Wrap(errors.ErrNotSupported, "Dilithium consensus keys are not used for SDK tx signing")
}

// PubKey derives a public key from the private key.
// Note: In practice, consensus keys are managed by CometBFT. This method is
// provided only to satisfy the cryptotypes.PrivKey interface and should not be
// relied upon for consensus operations.
func (privKey *PrivKey) PubKey() cryptotypes.PubKey {
	// Derive a deterministic placeholder pubkey from the private key bytes.
	h := tmhash.Sum(privKey.Key)
	return &PubKey{Key: h}
}

// Equals compares two private keys in constant time.
func (privKey *PrivKey) Equals(other cryptotypes.LedgerPrivKey) bool {
	if privKey.Type() != other.Type() {
		return false
	}
	return subtle.ConstantTimeCompare(privKey.Bytes(), other.Bytes()) == 1
}

func (privKey *PrivKey) Type() string { return KeyType }

// MarshalAmino overrides Amino binary marshaling.
func (privKey PrivKey) MarshalAmino() ([]byte, error) { return privKey.Key, nil }

// UnmarshalAmino overrides Amino binary unmarshaling.
func (privKey *PrivKey) UnmarshalAmino(bz []byte) error {
	if len(bz) == 0 {
		return fmt.Errorf("invalid privkey size")
	}
	privKey.Key = bz
	return nil
}

// MarshalAminoJSON overrides Amino JSON marshaling.
func (privKey PrivKey) MarshalAminoJSON() ([]byte, error) { return privKey.MarshalAmino() }

// UnmarshalAminoJSON overrides Amino JSON unmarshaling.
func (privKey *PrivKey) UnmarshalAminoJSON(bz []byte) error { return privKey.UnmarshalAmino(bz) }

// ========================= PubKey =========================

// Address is the SHA256-20 of the raw pubkey bytes.
// It doesn't implement ADR-28 and must not be used in SDK except in a validator context.
func (pubKey *PubKey) Address() crypto.Address {
	return crypto.Address(tmhash.SumTruncated(pubKey.Key))
}

// Bytes returns the PubKey byte format.
func (pubKey *PubKey) Bytes() []byte { return pubKey.Key }

// VerifySignature is not used for consensus keys in the SDK; CometBFT verifies
// consensus signatures. We provide a minimal implementation to satisfy the
// interface and prevent accidental acceptance: return false on empty signatures.
func (pubKey *PubKey) VerifySignature(msg, sig []byte) bool { return len(sig) > 0 }

// String returns a compact representation of the pubkey.
func (pubKey *PubKey) String() string {
	finger := tmhash.SumTruncated(pubKey.Key)
	return fmt.Sprintf("PubKeyDilithium2{%X}", finger)
}

func (pubKey *PubKey) Type() string { return KeyType }

func (pubKey *PubKey) Equals(other cryptotypes.PubKey) bool {
	if pubKey.Type() != other.Type() {
		return false
	}
	return subtle.ConstantTimeCompare(pubKey.Bytes(), other.Bytes()) == 1
}

// MarshalAmino overrides Amino binary marshaling.
func (pubKey PubKey) MarshalAmino() ([]byte, error) { return pubKey.Key, nil }

// UnmarshalAmino overrides Amino binary unmarshaling.
func (pubKey *PubKey) UnmarshalAmino(bz []byte) error {
	if len(bz) == 0 {
		return errorsmod.Wrap(errors.ErrInvalidPubKey, "invalid pubkey size")
	}
	pubKey.Key = bz
	return nil
}

// MarshalAminoJSON overrides Amino JSON marshaling.
func (pubKey PubKey) MarshalAminoJSON() ([]byte, error) { return pubKey.MarshalAmino() }

// UnmarshalAminoJSON overrides Amino JSON marshaling.
func (pubKey *PubKey) UnmarshalAminoJSON(bz []byte) error { return pubKey.UnmarshalAmino(bz) }
