# Dilithium Integration – Step 2: HD Derivation + Keyring Support

This step enables the Cosmos SDK keyring to create and store Dilithium2 keys via the CLI, and wires a basic HD derivation path for deterministically generating Dilithium private keys from a mnemonic.

## Files Modified

- `crypto/hd/algo.go`
  - Added new HD algorithm type and implementation for Dilithium2:
    - `const Dilithium2Type = "dilithium2"`
    - `var Dilithium2 = dilithium2Algo{}`
    - `dilithium2Algo` implements `Name()`, `Derive()`, and `Generate()`.
  - `Derive()` reuses the standard BIP39 seed + BIP32 path derivation used by secp256k1 to produce deterministic bytes.
  - `Generate()` expands the derived bytes to Dilithium private key size and returns `*dilithium.PrivKey`.

- `crypto/keys/dilithium/dilithium.go`
  - Defined sizes aligned with your CometBFT Dilithium2 implementation:
    - `const PubKeySize = 1312`
    - `const PrivKeySize = 2528`
  - Updated `PrivKey.PubKey()` to produce a deterministic 1312-byte `PubKey` from a hash pattern to satisfy interfaces (placeholder; not cryptographic).

- `crypto/keyring/keyring.go`
  - Included the new algorithm in the default supported software algorithms:
    - `SupportedAlgos: SigningAlgoList{hd.Secp256k1, hd.Dilithium2}`
  - Ledger algorithms remain unchanged (Dilithium is software-only here).

## Generated Code

- No new generated files in this step. `make build` succeeds after these changes.

## Usage (CLI)

Create a Dilithium2 key using the test backend (non-interactive password):

```bash
simd keys add test-dili --key-type dilithium2 --keyring-backend test --output json
```

Inspect keys:

```bash
simd keys list --keyring-backend test
simd keys show test-dili --keyring-backend test --output json
```

You should observe the Any type URL `/cosmos.crypto.dilithium.PubKey` in the JSON output.

## Notes and Limitations

- Signing transactions with Dilithium keys is not supported in this SDK layer. `PrivKey.Sign()` returns `ErrNotSupported` to make this explicit. Use Dilithium keys for consensus/offline contexts only, unless full tx signing support is added end-to-end.
- The `PrivKey.PubKey()` derivation is a deterministic placeholder for interface compliance and tests; it is not a real Dilithium key derivation.
- For consensus-level use, CometBFT must understand Dilithium keys in `tendermint.crypto.PublicKey`. See Step 3 notes below. We will wire Cosmos SDK conversions to the exact oneof field exposed by your CometBFT fork (e.g. `Dilithium2` or `Dilithium`).

## Next Steps (Step 3 Preview)

- Integrate Dilithium in the CometBFT layer:
  - Ensure `tendermint/crypto/keys.proto` includes a Dilithium oneof case in your CometBFT fork.
  - Update `crypto/codec/cmt.go` conversions to/from CometBFT’s `PublicKey` for Dilithium.
  - Rebuild and test genesis flows involving Dilithium validator keys.

If you want automated verification scripts for Step 2 (key creation/list/show), we can add them under `scripts/` and reference them here.
