# Validator Keys and Linking (Beginner Guide)

This guide explains, in simple terms, how a validator’s two keypairs work, when each is created, how they are linked on-chain, how to add new validators, and how rewards flow. It also gives copy-paste commands to verify everything on your machine.

---

## TL;DR: Two keys, two jobs
- **Operator (wallet) key**
  - Signs normal transactions: bank sends, create-validator, delegate, withdraw rewards, governance.
  - Algorithm: `secp256k1`.
  - Addresses (same key, different prefixes):
    - Account: `cosmos1...`
    - Operator: `cosmosvaloper1...`
  - Lives in the **keyring** (managed by `simd keys ...`).

- **Consensus (engine) key**
  - Signs consensus messages: prevote, precommit, commit (i.e., block production).
  - Algorithm here: `Dilithium2` (post-quantum).
  - Address: `cosmosvalcons1...` (derived from the consensus pubkey).
  - Lives in `~/.simapp/config/priv_validator_key.json` (managed by CometBFT; not in the keyring).

These two keypairs are independent and not derived from each other. They are linked on-chain by a staking message.

---

## When each key is created
- **Node init (before chain start)**
  - The node generates the **consensus key** and writes it to `~/.simapp/config/priv_validator_key.json`.
  - In this build, the consensus algorithm defaults to Dilithium2 and genesis enforces `pub_key_types = ["dilithium2"]`.

- **Wallet creation (any time)**
  - `simd keys add ...` creates a **secp256k1** account key in the keyring. This is your human-operated wallet for transactions.

---

## How they get linked (the Create-Validator message)
- You, the operator, send `MsgCreateValidator` (during genesis as a gentx or later on a running chain).
- You **sign** the tx with your operator/wallet key (secp256k1).
- Inside the message, you **embed** the node’s **consensus public key** (Dilithium).
- The staking module stores a mapping in state:
  - `operator (cosmosvaloper1...) → consensus_pubkey (Any: /cosmos.crypto.dilithium.PubKey)`
  - It also derives and stores the `consensus address (cosmosvalcons1...)`.
- From that point on:
  - Your operator wallet continues to sign transactions.
  - Your node uses the Dilithium consensus key to sign votes and blocks automatically.

Mentally, `MsgCreateValidator` looks like this (simplified):
```json
{
  "@type": "/cosmos.staking.v1beta1.MsgCreateValidator",
  "validator_address": "cosmosvaloper1...",
  "delegator_address": "cosmos1...",
  "pubkey": {"@type": "/cosmos.crypto.dilithium.PubKey", "key": "<base64-1312>"},
  "value": {"denom": "stake", "amount": "1000000"}
}
```
- The tx is signed by your secp256k1 operator key (in `auth_info.signer_infos`).

---

## Where this is wired in the SDK (for reference)
- `x/genutil/client/cli/init.go`: defaults the consensus key algo to Dilithium2 and writes genesis params enforcing Dilithium.
- `x/genutil/client/cli/gentx.go`: builds the gentx/`MsgCreateValidator`, loads the consensus pubkey from `priv_validator_key.json`, embeds it in the message, and signs the tx with your operator wallet.
- `crypto/codec/cmt.go`: converts between CometBFT `PublicKey_Dilithium` and SDK `dilithium.PubKey` (Protobuf Any).
- `crypto/keys/dilithium/dilithium.go`: defines Dilithium key types, sizes, and `PubKey.Address()`.
- `proto/cosmos/crypto/dilithium/keys.proto`: protobuf definitions for `/cosmos.crypto.dilithium.PubKey` and `PrivKey`.

---

## Quick verification commands (no variables)
Run from repo root. These commands assume the default home `~/.simapp` and the `test` keyring backend.

- **List your wallet keys (operator/user accounts)**
```bash
./build/simd keys list --keyring-backend test
```

- **Show your operator and account addresses**
```bash
./build/simd keys show mykey -a --keyring-backend test
./build/simd keys show mykey --bech val -a --keyring-backend test
```

- **Show the node’s consensus key (Dilithium) and address**
```bash
./build/simd tendermint show-validator
./build/simd tendermint show-address
jq '.pub_key' "$HOME/.simapp/config/priv_validator_key.json"
```

- **See the on-chain validator mapping (operator → consensus_pubkey)**
```bash
./build/simd query staking validator $(./build/simd keys show mykey --bech val -a --keyring-backend test) -o json | jq '.validator | {operator_address, consensus_pubkey, consensus_address}'
```

- **When the node is running, confirm Dilithium via RPC**
```bash
curl -s http://localhost:26657/validators | jq '.result.validators[0].pub_key.type'
```

---

## Starting a fresh single-validator chain (end-to-end)
WARNING: This wipes local state.
```bash
make build
rm -rf "$HOME/.simapp"
./build/simd init mynode --chain-id pqc-chain
./build/simd keys add mykey --keyring-backend test
./build/simd keys add user1 --keyring-backend test
./build/simd genesis add-genesis-account $(./build/simd keys show mykey -a --keyring-backend test) 100000000stake
./build/simd genesis add-genesis-account $(./build/simd keys show user1 -a --keyring-backend test) 100000000stake
./build/simd genesis gentx mykey 1000000stake --chain-id pqc-chain --keyring-backend test
./build/simd genesis collect-gentxs
./build/simd genesis validate
./build/simd start
```

- After start, verify mapping and Dilithium usage using the commands above.

---

## Adding a second validator to a running chain (conceptual)
On the new validator’s machine:
- Initialize the node (this creates its own Dilithium consensus key in its `priv_validator_key.json`).
- Create a wallet key in the keyring (secp256k1), fund it from the network with enough tokens.
- Submit `MsgCreateValidator` from that wallet, embedding the node’s Dilithium pubkey.
- The chain stores the operator → consensus_pubkey link and activates the validator once bonded.

Why Dilithium is enforced: genesis `consensus.params.validator.pub_key_types` is set to `["dilithium2"]`, so non-Dilithium consensus keys are rejected.

---

## Rewards: where they go and how to use them
- Rewards are tracked by the distribution module. They accrue to your validator/delegators.
- By default, your rewards withdraw address is your account (wallet) address.
- To **use** rewards, you submit a withdraw transaction with your wallet. Once withdrawn, rewards land in your bank balance and can be spent or transferred.

Common actions (no variables):
```bash
./build/simd tx distribution withdraw-rewards $(./build/simd keys show mykey --bech val -a --keyring-backend test) --commission --from mykey --chain-id pqc-chain --keyring-backend test -y --gas auto --fees 2000stake --broadcast-mode block
./build/simd tx distribution set-withdraw-addr $(./build/simd keys show user1 -a --keyring-backend test) --from mykey --chain-id pqc-chain --keyring-backend test -y --gas auto --fees 2000stake --broadcast-mode block
```

Transfers between accounts (no variables):
```bash
./build/simd tx bank send mykey $(./build/simd keys show user1 -a --keyring-backend test) 250000stake --chain-id pqc-chain --keyring-backend test -y --gas auto --fees 2000stake --broadcast-mode block
./build/simd tx bank send user1 $(./build/simd keys show mykey -a --keyring-backend test) 100000stake --chain-id pqc-chain --keyring-backend test -y --gas auto --fees 2000stake --broadcast-mode block
```

---

## Troubleshooting
- "secp256k1" in `keys add`: expected; that’s your wallet key for tx signing.
- Validator shows Dilithium in `tendermint show-validator` and `/validators`: expected; that’s the consensus key used by the node.
- Genesis parse errors with validator keys: ensure `genesis.json` has `"validator.pub_key_types": ["dilithium2"]` and your validator pubkey decodes to 1312 bytes.

---

## Glossary
- **Operator key**: wallet key (secp256k1) that controls the validator and signs transactions.
- **Consensus key**: node engine key (Dilithium2) that signs consensus messages. Stored in `priv_validator_key.json`.
- **MsgCreateValidator**: staking message that links operator → consensus_pubkey on-chain.
- **Consensus address**: bech32 address derived from the consensus pubkey: `cosmosvalcons1...`.

---

## Why we keep tx signing on secp256k1
- Wallet tooling and Ledger support are widely available for secp256k1.
- This integration makes the consensus layer post-quantum (Dilithium2) while keeping the user-facing transaction layer familiar and compatible.
