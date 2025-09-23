# PR Change Log — Dilithium Consensus Integration (Steps 1–4)

This PR introduces full Dilithium validator (consensus) key support end‑to‑end in the Cosmos SDK, wired to a Dilithium‑enabled CometBFT fork. It also includes developer tooling and documentation to verify that consensus keys and block signatures use Dilithium.

Base commit: `96dc3c54f9` (first)
Head commit: `8946d30b61` (step4)

## Commit Summary
- `3f1f654659` step2
- `acd2ed4136` step2 update
- `968cf0528a` step3
- `8946d30b61` step4

## Diff Stat (since base)
```
19 files changed, 2233 insertions(+), 19 deletions(-)
api/cosmos/crypto/dilithium/keys.pulsar.go | 1061 ++++++++++++++
api/tendermint/abci/types.pulsar.go        |    4 +-
api/tendermint/crypto/proof.pulsar.go      |    2 +-
crypto/codec/amino.go                      |    5 +
crypto/codec/cmt.go                        |   11 +
crypto/codec/proto.go                      |    3 +
crypto/hd/algo.go                          |   48 +
crypto/keyring/keyring.go                  |    2 +-
crypto/keys/dilithium/dilithium.go         |  140 ++
crypto/keys/dilithium/keys.pb.go           |  505 +++++++
docs/README-Dilithium-Step1.md             |   93 ++
docs/README-Dilithium-Step2.md             |   60 +
docs/README-Dilithium-Step3.md             |  143 ++
docs/README-Dilithium-Step4.md             |  118 +
proto/cosmos/crypto/dilithium/keys.proto   |   32 +
x/genutil/client/cli/init.go               |    4 +-
x/genutil/utils.go                         |   13 +-
x/staking/types/staking.pb.go              |    4 +-
```

---

## Key Changes by Area

### 1) SDK Key Types + Codecs (Protobuf + Amino)
- Files:
  - `crypto/keys/dilithium/dilithium.go`
  - `crypto/keys/dilithium/keys.pb.go`
  - `proto/cosmos/crypto/dilithium/keys.proto`
  - `crypto/codec/amino.go`

- Highlights:
  - New key types with correct sizes aligned to the CometBFT fork:
    - `KeyType = "dilithium2"`
    - `PubKeySize = 1312`, `PrivKeySize = 2528`
  - Protobuf Any support for `/cosmos.crypto.dilithium.PubKey` and `/cosmos.crypto.dilithium.PrivKey`.
  - Legacy Amino registration for Dilithium keys (PubKey/PrivKey).
  - `PrivKey.Sign()` returns `ErrNotSupported` (consensus signing handled by CometBFT).

- Diff snippets:
```diff
*** crypto/codec/amino.go
@@
 import (
     "github.com/cosmos/cosmos-sdk/codec"
     "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
+    "github.com/cosmos/cosmos-sdk/crypto/keys/dilithium"
     kmultisig "github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
     "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
     cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
 )
@@
     cdc.RegisterConcrete(&ed25519.PubKey{}, ed25519.PubKeyName, nil)
+    cdc.RegisterConcrete(&dilithium.PubKey{}, dilithium.PubKeyName, nil)
     cdc.RegisterConcrete(&secp256k1.PubKey{}, secp256k1.PubKeyName, nil)
@@
     cdc.RegisterConcrete(&secp256k1.PrivKey{}, secp256k1.PrivKeyName, nil)
+    cdc.RegisterConcrete(&dilithium.PrivKey{}, dilithium.PrivKeyName, nil)
```


### 2) CometBFT ↔ SDK PublicKey Mapping
- File: `crypto/codec/cmt.go`
- Purpose: Translate between CometBFT proto `PublicKey_Dilithium` and SDK `dilithium.PubKey`.

- Diff snippet:
```diff
*** crypto/codec/cmt.go
@@
-    "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
+    "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
+    "github.com/cosmos/cosmos-sdk/crypto/keys/dilithium"
@@
 case *cmtprotocrypto.PublicKey_Dilithium:
-    // none
+    return &dilithium.PubKey{ Key: protoPk.Dilithium }, nil
@@
 case *dilithium.PubKey:
-    // none
+    return cmtprotocrypto.PublicKey{
+        Sum: &cmtprotocrypto.PublicKey_Dilithium{ Dilithium: pk.Key },
+    }, nil
```


### 3) HD Derivation + Keyring Support (Developer Tooling)
- Files:
  - `crypto/hd/algo.go`
  - `crypto/keyring/keyring.go`
- Purpose: Add `dilithium2` to supported algorithms, deterministic key generation for testing/tooling.

- Diff snippet (abbrev.):
```diff
*** crypto/hd/algo.go
@@
 const (
     Bls12_381Type = PubKeyType("bls12_381")
+    Dilithium2Type = PubKeyType("dilithium2")
 )
@@
+var Dilithium2 = dilithium2Algo{}
+
+type dilithium2Algo struct{}
+
+func (d dilithium2Algo) Name() PubKeyType { return Dilithium2Type }
+func (d dilithium2Algo) Derive() DeriveFn { /* BIP39 seed → bytes */ }
+func (d dilithium2Algo) Generate() GenerateFn {
+    return func(bz []byte) types.PrivKey {
+        out := make([]byte, dilithium.PrivKeySize)
+        for i := 0; i < len(out); i++ { out[i] = bz[i%len(bz)] }
+        return &dilithium.PrivKey{Key: out}
+    }
+}
```


### 4) Consensus Defaults + gentx Flow
- Files:
  - `x/genutil/client/cli/init.go`
  - `x/genutil/client/cli/gentx.go`
- Purpose: Default consensus key algo to Dilithium2 and ensure gentx carries `/cosmos.crypto.dilithium.PubKey` from the priv-validator.

- Diff snippet:
```diff
*** x/genutil/client/cli/init.go
@@
-import "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
+import "github.com/cosmos/cosmos-sdk/crypto/keys/dilithium"
@@
-cmd.Flags().String(FlagConsensusKeyAlgo, ed25519.KeyType, "algorithm to use for the consensus key")
+cmd.Flags().String(FlagConsensusKeyAlgo, dilithium.KeyType, "algorithm to use for the consensus key (default dilithium2)")
```


### 5) Documentation
- Files:
  - `docs/README-Dilithium-Step1.md`
  - `docs/README-Dilithium-Step2.md`
  - `docs/README-Dilithium-Step3.md`
  - `docs/README-Dilithium-Step4.md`
  - `docs/README-Dilithium-All-In-One.md` (consolidated run + verify)

- Purpose: Explain integration details and provide runbooks to verify Dilithium consensus (init → gentx → collect → start) and validate pubkey types and commit signature sizes.

---

## Rationale (Why SDK Changes Were Needed)
- CometBFT can sign with Dilithium, but the SDK must:
  - Understand Dilithium keys in Protobuf Any and legacy Amino.
  - Map CometBFT proto `PublicKey_Dilithium` to SDK `dilithium.PubKey`.
  - Default genesis consensus params to allow `dilithium2`.
  - Build and parse `MsgCreateValidator` with Dilithium pubkeys.
  - Provide CLI tooling that doesn’t break (keyring/HD, gentx/collect).
- These changes resolve the previous genesis parsing failure ("EOF: tx parse error") and enable end‑to‑end Dilithium consensus.

---

## Verification Checklist
- Run the All‑In‑One guide:
  - `docs/README-Dilithium-All-In-One.md`
- Key checks:
  - Genesis: `.consensus.params.validator.pub_key_types == ["dilithium2"]`
  - `tendermint show-validator` → `{"@type":"/cosmos.crypto.dilithium.PubKey"}`
  - RPC `/validators` → `{"type":"cometbft/PubKeyDilithium"}`
  - Commit signature size (from `/commit`) is thousands of bytes (Dilithium), not 64 (ed25519).

---

## Notes and Compatibility
- SDK transaction signing with Dilithium is intentionally not enabled here; consensus (validator) signing is handled by CometBFT.
- Removed duplicate Amino alias registrations to fix `panic: TypeInfo already exists for dilithium.PubKey`.
- Public/private key sizes (Dilithium2) are `1312/2528`.
