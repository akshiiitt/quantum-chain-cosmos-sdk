# Dilithium Integration – Step 4: End-to-End Verification

This step verifies the full Dilithium consensus flow end-to-end with your Dilithium-enabled CometBFT fork:

- Initialize a chain configured to accept Dilithium2 validator keys
- Create a Dilithium validator gentx
- Collect and finalize genesis
- Start the node and produce blocks

The commands below use a temporary home directory so they won’t touch any existing setup.

## Prerequisites

- Built binary at `./build/simd`
- CometBFT fork replace is set in `go.mod` (already done)

## 1) Initialize a fresh node

```bash
H=/tmp/pqc-simd
./build/simd init mynode --chain-id pqc-chain --home "$H"
```

Verify consensus params allow Dilithium2:

```bash
jq '.consensus.params.validator.pub_key_types' "$H/config/genesis.json"
# Expected: ["dilithium2"] (AppGenesis format uses .consensus.params)
```

## 2) Inspect validator public key

```bash
./build/simd tendermint show-validator --home "$H" | tee /tmp/val.json
jq -r '.type' /tmp/val.json     # Expected: cometbft/PubKeyDilithium
jq -r '.value' /tmp/val.json | base64 -d | wc -c  # Expected: 1312
```

## 3) Create a key, fund it, and generate a Dilithium gentx

```bash
./build/simd keys add mykey --keyring-backend test --home "$H"
ADDR=$(./build/simd keys show mykey -a --keyring-backend test --home "$H")
./build/simd genesis add-genesis-account "$ADDR" 100000000stake --home "$H"
./build/simd genesis gentx mykey 1000000stake --chain-id pqc-chain --keyring-backend test --home "$H"
```

If you prefer a custom `validator.json`, both of the following pubkey forms are accepted:

- Protobuf Any form:

```json
{
  "pubkey": {"@type":"/cosmos.crypto.dilithium.PubKey","key":"<base64-1312>"},
  "amount": "1000000stake",
  "moniker": "myvalidator",
  "commission-rate": "0.1",
  "commission-max-rate": "0.2",
  "commission-max-change-rate": "0.01",
  "min-self-delegation": "1"
}
```

- Amino alias form (also supported):

```json
{
  "pubkey": {"type":"cometbft/PubKeyDilithium","value":"<base64-1312>"},
  "amount": "1000000stake",
  "moniker": "myvalidator",
  "commission-rate": "0.1",
  "commission-max-rate": "0.2",
  "commission-max-change-rate": "0.01",
  "min-self-delegation": "1"
}
```

## 4) Collect gentxs and finalize genesis

```bash
./build/simd genesis collect-gentxs --home "$H"
```

## 5) Start the node

```bash
./build/simd start --home "$H"
```

Expected behavior:

- Node starts without `EOF: tx parse error`
- Blocks are produced normally

Optional status checks (in another terminal):

```bash
curl -s http://localhost:26657/status | jq '.result.validator_info.pub_key.type'
# Expected: "cometbft/PubKeyDilithium"

curl -s http://localhost:26657/validators | jq '.result.validators[0].pub_key.type'
# Expected: "cometbft/PubKeyDilithium"
```

## Troubleshooting

- If you see a parse error at genesis:
  - Ensure `crypto/codec/cmt.go` contains Dilithium mappings
  - Verify `genesis.json` -> `.consensus_params.validator.pub_key_types == ["dilithium2"]`
  - Ensure validator pubkey JSON decodes to 1312 bytes

- If keys show Ed25519 unexpectedly:
  - Make sure you initialized with the new binary and the CometBFT fork replace is active

## Notes

- SDK transaction signing with Dilithium is intentionally not enabled. Dilithium is used for consensus keys here.
- Public/private key sizes used by this integration are 1312/2528 bytes (Dilithium2).
