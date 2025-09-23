# Dilithium Integration – Step 3: Consensus Integration (CometBFT)

This step wires the Cosmos SDK to your Dilithium-enabled CometBFT so that validators can use Dilithium2 public keys throughout genesis and runtime. With these changes, you can initialize a chain that enforces `dilithium2` validator keys and successfully process genesis/gentx transactions without parsing errors.

## What’s included in Step 3

- CometBFT <-> Cosmos SDK key mapping
  - `crypto/codec/cmt.go` now maps CometBFT Dilithium public keys to the SDK and back:
    - From CometBFT proto to SDK: `PublicKey_Dilithium` -> `dilithium.PubKey`
    - From SDK to CometBFT proto: `dilithium.PubKey` -> `PublicKey_Dilithium`

- Default consensus key algorithm in init
  - `x/genutil/client/cli/init.go` sets the default `--consensus-key-algo` to `dilithium2`.
  - Genesis `consensus_params.validator.pub_key_types` will default to `["dilithium2"]` when using `simd init`.

- Priv-Validator generation uses CometBFT fork’s default
  - `x/genutil/utils.go` always uses `privval.LoadOrGenFilePV(...)` so the Dilithium-enabled CometBFT fork generates Dilithium keys.
  - The Ed25519-from-mnemonic branch is removed to avoid consensus key type mismatch.

- Amino JSON aliases for CometBFT types
  - `crypto/keys/dilithium/dilithium.go` adds aliases:
    - `PubKeyNameAlias = "cometbft/PubKeyDilithium"`
    - `PrivKeyNameAlias = "cometbft/PrivKeyDilithium"`
  - `crypto/codec/amino.go` registers both canonical names (tendermint/...) and aliases (cometbft/...) so JSON produced by `tendermint show-validator` parses correctly when fed back to SDK.

- Dilithium sizes aligned with your fork
  - `crypto/keys/dilithium/dilithium.go`
    - `PubKeySize = 1312`
    - `PrivKeySize = 2528`

## Prerequisites

- Your `go.mod` is configured to use the Dilithium-enabled CometBFT fork. Example (already set):

```
replace github.com/cometbft/cometbft => github.com/akshiiitt/CometBFT-Quantum-Dilithium2- v0.0.0-20250918073650-14e5f9019929
```

- The fork’s CometBFT proto oneof must include a Dilithium field named exactly like we map in `cmt.go` (e.g. `PublicKey_Dilithium`).

## Quick verification

1) Build SDK

```
make build
```

2) Initialize a chain with Dilithium consensus keys

```
./build/simd init mynode --chain-id pqc-chain
```

- Check consensus params in genesis:

```
cat ~/.simapp/config/genesis.json | jq '.consensus_params.validator.pub_key_types'
# Expected: ["dilithium2"]
```

3) Confirm validator key type and size

```
./build/simd tendermint show-validator | tee /tmp/val.json
jq -r '.type' /tmp/val.json   # Expected: cometbft/PubKeyDilithium
jq -r '.value' /tmp/val.json | base64 -d | wc -c  # Expected: 1312
```

4) Fund the account and create a gentx

```
./build/simd keys add mykey --keyring-backend test
ADDR=$(./build/simd keys show mykey -a --keyring-backend test)
./build/simd genesis add-genesis-account "$ADDR" 100000000stake
./build/simd genesis gentx mykey 1000000stake --chain-id pqc-chain --keyring-backend test
./build/simd genesis collect-gentxs
```

5) Start the chain

```
./build/simd start
```

- You should see blocks being produced. No `EOF: tx parse error` should occur during genesis processing.

## Using validator.json directly (alternative)

If you prefer `create-validator` with an explicit file, both of the following pubkey forms are accepted:

- Protobuf Any form (preferred)

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

- Amino alias form (supported via alias registration)

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

## Notes on testnet command

The `simapp simd testnet` path bootstraps genesis via different helpers. Ensure your CometBFT fork’s default consensus params include `dilithium2`. If not, either:

- Post-edit the generated `genesis.json` to set `consensus_params.validator.pub_key_types: ["dilithium2"]`, or
- Initialize with `simd init` first (which sets the key type), then proceed with your testnet setup.

## Troubleshooting

- EOF parse error during genesis
  - Ensure `crypto/codec/cmt.go` includes Dilithium mappings and you’re on the correct CometBFT fork version.
  - Verify genesis has `pub_key_types: ["dilithium2"]`.
  - Ensure the validator pubkey JSON uses either `/cosmos.crypto.dilithium.PubKey` or the `cometbft/PubKeyDilithium` alias and that the base64 decodes to 1312 bytes.

- Key size mismatch
  - Confirm your CometBFT fork and SDK sizes align (PubKey 1312 bytes, PrivKey 2528 bytes). Re-check the base64 payload length.

## Limitations

- SDK transaction signing with Dilithium is not enabled. `PrivKey.Sign()` returns `ErrNotSupported`. Dilithium is used for consensus keys.

## Status

Step 3 is complete. The SDK is fully wired to the Dilithium-enabled CometBFT fork for consensus keys, with default genesis configuration, codec mappings, and JSON compatibility in place. Follow the verification steps above to validate on your machine.
