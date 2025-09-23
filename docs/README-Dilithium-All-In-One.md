# Dilithium Integration: All‑In‑One Guide (Commands + Changes)

This guide gives you a complete, start‑to‑end command sequence to run a Dilithium‑enabled Cosmos SDK chain and verify that validator consensus keys and signatures are using Dilithium. It also explains the SDK changes made in Steps 1–4 and why they were necessary even after replacing CometBFT.

Repository root: `/data/Akshit/Cosmos-Quantum/cosmos-sdk`

---

## 1) Quick Start Commands (default home: `~/.simapp`)

Run all commands from the repo root. These use the default home; no temp paths.

- Build the binary
```bash
make build
```

- Initialize the chain (Dilithium2 is the default consensus key type)
```bash
./build/simd init mynode --chain-id pqc-chain
```

- Verify genesis enforces Dilithium validators
```bash
jq '.consensus.params.validator.pub_key_types' "$HOME/.simapp/config/genesis.json"
# Expected: ["dilithium2"]
```

- Verify your validator public key is Dilithium (SDK + CometBFT views)
```bash
# SDK Any view
./build/simd tendermint show-validator
# Expected: {"@type":"/cosmos.crypto.dilithium.PubKey","key":"..."}

# Check Dilithium pubkey size (Dilithium2 = 1312 bytes)
./build/simd tendermint show-validator | jq -r '.key' | base64 -d | wc -c
# Expected: 1312

# CometBFT priv-validator file (amino JSON)
jq '.pub_key' "$HOME/.simapp/config/priv_validator_key.json"
# Expected: {"type":"cometbft/PubKeyDilithium","value":"..."}
```

- Create a user key and fund it in genesis (account keys stay secp256k1)
```bash
./build/simd keys add mykey --keyring-backend test
ADDR=$(./build/simd keys show mykey -a --keyring-backend test)
./build/simd genesis add-genesis-account "$ADDR" 100000000stake

-----------------

./build/simd keys add mykey --keyring-backend test 

./build/simd genesis add-genesis-account "$(./build/simd keys show mykey -a --keyring-backend test)" 100000000stake
```

- Create a Dilithium validator gentx (pulls pubkey from priv-validator)
```bash
./build/simd genesis gentx mykey 1000000stake --chain-id pqc-chain --keyring-backend test
```

- Verify gentx contains a Dilithium pubkey
```bash
cat "$HOME/.simapp/config/gentx/"gentx-*.json | jq '.body.messages[0].pubkey'
# Expected: {"@type":"/cosmos.crypto.dilithium.PubKey","key":"..."}
```

- Collect gentxs (and optionally validate genesis)
```bash
./build/simd genesis collect-gentxs
./build/simd genesis validate
# No errors expected
```

- Start the node
```bash
./build/simd start
```

- In another terminal, verify via RPC that the active validator uses Dilithium
```bash
curl -s http://localhost:26657/validators | jq '.result.validators[0].pub_key'
# Expected: {"type":"cometbft/PubKeyDilithium","value":"..."}
```

- Prove blocks are signed with Dilithium (commit signatures are large)
```bash
HEIGHT=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')
echo "HEIGHT=$HEIGHT"

curl -s "http://localhost:26657/commit?height=${HEIGHT}" \
| jq -r '.result.signed_header.commit.signatures[0].signature' \
| base64 -d | wc -c
# Expected: large (thousands of bytes), not 64. This indicates a Dilithium signature rather than ed25519.

# Cross-check validator pubkey type again
curl -s http://localhost:26657/validators | jq -r '.result.validators[0].pub_key.type'
# Expected: cometbft/PubKeyDilithium
```

- Reset (optional, wipes local chain state)
```bash
# Stop simd (Ctrl+C), then:
rm -rf "$HOME/.simapp"
```

---

## 2) What Changed in the SDK (Steps 1–4) and Why

Replacing CometBFT with a Dilithium‑capable fork lets the consensus engine sign blocks with Dilithium, but the SDK must still understand Dilithium keys everywhere they appear: Protobuf Any, legacy Amino, gentx/genesis, CLI flows, and consensus params. These SDK changes close that gap.

### Step 1 — Protobuf + Codec Registration (SDK learns Dilithium)
- Files:
  - `crypto/keys/dilithium/dilithium.go`
  - `crypto/keys/dilithium/keys.pb.go`
  - `crypto/codec/amino.go`
- Key points:
  - Implemented `dilithium.PrivKey` and `dilithium.PubKey` with:
    - `KeyType = "dilithium2"`
    - Amino names: `tendermint/PrivKeyDilithium2`, `tendermint/PubKeyDilithium2`
    - Sizes aligned to your fork: `PubKeySize = 1312`, `PrivKeySize = 2528`
  - Registered Dilithium in Protobuf interfaces and legacy Amino so `/cosmos.crypto.dilithium.PubKey` encodes/decodes correctly.
  - `PrivKey.Sign()` returns `ErrNotSupported` (consensus signing is performed by CometBFT, not by SDK).
- Why it matters:
  - Without this, `MsgCreateValidator` carrying a Dilithium pubkey in Any fails to decode, causing the genesis parse error.

### Step 2 — Keyring + HD Derivation (Deterministic, Sized Keys)
- Files:
  - `crypto/hd/algo.go`
  - `crypto/keyring/keyring.go`
  - `crypto/keys/dilithium/dilithium.go`
- Key points:
  - Added `dilithium2Algo.Generate()` to produce deterministic, properly‑sized private keys for tooling/tests.
  - `PrivKey.PubKey()` derives a deterministic pubkey (for interface satisfaction and keyring ops).
  - Included Dilithium in supported keyring algorithms so CLI `keys add/show/list` works.
- Why it matters:
  - Enables developer workflows and avoids “pubkey incorrect size” / all‑zero material issues.

### Step 3 — Consensus Integration (CometBFT ↔ SDK)
- Files:
  - `crypto/codec/cmt.go`
  - `x/genutil/client/cli/init.go`
  - `x/genutil/client/cli/gentx.go`
  - `crypto/codec/amino.go` (panic fix)
- Key points:
  - Mapped CometBFT `PublicKey_Dilithium` ↔ SDK `dilithium.PubKey` in `cmt.go`.
  - Defaulted `simd init` to Dilithium2 consensus key type and ensured genesis param `consensus.params.validator.pub_key_types == ["dilithium2"]`.
  - `genesis gentx` reads the Dilithium pubkey from `priv_validator_key.json` and embeds it as `/cosmos.crypto.dilithium.PubKey` in `MsgCreateValidator`.
  - Removed duplicate Amino registrations that previously caused `panic: TypeInfo already exists for dilithium.PubKey`.
- Why it matters:
  - Eliminates the “EOF: tx parse error” and lets gentx/collect/start work with Dilithium.

### Step 4 — End‑to‑End Verification & Documentation
- Files:
  - `docs/README-Dilithium-Step3.md` (consensus integration notes)
  - `docs/README-Dilithium-Step4.md` (verification flow)
  - This file: `docs/README-Dilithium-All-In-One.md`
- Key points:
  - Provides a tested workflow (init → gentx → collect → start) and concrete checks for Dilithium pubkeys and signature sizes via RPC.

---

## 3) Where Dilithium Is Used vs Not Used
- Consensus keys (validators):
  - Managed by CometBFT in `~/.simapp/config/priv_validator_key.json` with types `cometbft/PrivKeyDilithium` and `cometbft/PubKeyDilithium`.
  - Enforced by `genesis.json` at `.consensus.params.validator.pub_key_types == ["dilithium2"]`.
  - Used to sign blocks (precommits/commits). You see large commit signatures via `:26657/commit`.
- Account keys (wallets):
  - Remain secp256k1 by default for normal transactions.
  - Dilithium is not used for SDK tx signing in this setup (`PrivKey.Sign()` is unsupported by design here).

---

## 4) Troubleshooting
- Genesis parse error (e.g., `EOF: tx parse error`):
  - Ensure you’re on the Dilithium CometBFT fork.
  - Verify `crypto/codec/cmt.go` includes Dilithium mappings.
  - Check `genesis.json` → `.consensus.params.validator.pub_key_types == ["dilithium2"]`.
  - Confirm gentx pubkey is `/cosmos.crypto.dilithium.PubKey` and base64 decodes to 1312 bytes.
- Amino panic (`TypeInfo already exists for dilithium.PubKey`):
  - Remove duplicate Amino alias registrations for Dilithium in `crypto/codec/amino.go` (already fixed here).
- Validator shows ed25519 unexpectedly:
  - Re‑init with the current binary; ensure the fork replace is active and `genesis.json` enforces `dilithium2`.

---

## 5) References
- Code touched:
  - `crypto/keys/dilithium/dilithium.go`
  - `crypto/keys/dilithium/keys.pb.go`
  - `crypto/codec/cmt.go`
  - `crypto/codec/amino.go`
  - `crypto/hd/algo.go`
  - `crypto/keyring/keyring.go`
  - `x/genutil/client/cli/init.go`
  - `x/genutil/client/cli/gentx.go`
  - `simapp/simd/cmd/commands.go`
- Additional docs:
  - `docs/README-Dilithium-Step3.md`
  - `docs/README-Dilithium-Step4.md`
