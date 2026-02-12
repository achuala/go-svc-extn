# Crypto Package Usage Guide

## Overview

This package provides envelope encryption using [Google Tink](https://developers.google.com/tink).
The architecture separates keys into two layers:

```
Master Key (KEK)          - AES-256, lives ONLY in env var / K8s Secret
    |
    +-- encrypts -->  Keyset file (DEK)   - JSON file, safe to store in git/configmap
                          |
                          +-- encrypts -->  Application data
```

- **KEK (Key Encryption Key)**: A raw AES-256 key injected at runtime via environment variable. Never written to disk or config.
- **DEK (Data Encryption Key)**: A Tink keyset stored as an encrypted JSON file. Can be committed to git, stored in a ConfigMap, or mounted as a file. Useless without the KEK.

## 1. Bootstrap (One-Time Setup)

Run this once to generate your initial keys. This can be a standalone Go script,
a CI pipeline step, or a `go test` helper.

```go
package main

import (
    "fmt"
    "log"

    "github.com/achuala/go-svc-extn/pkg/crypto/encdec"
)

func main() {
    // Step 1: Generate a master key (KEK).
    // Store this in your secret manager (K8s Secret, Vault, etc.)
    masterKey, err := encdec.GenerateMasterKey()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("MASTER_KEY:", masterKey)

    // Step 2: Generate an encrypted keyset file (DEK).
    // This file is safe to store in git or a ConfigMap.
    masterAEAD, err := encdec.NewLocalAEAD(masterKey)
    if err != nil {
        log.Fatal(err)
    }

    ad := []byte("my-service keyset")  // associated data - must match at decrypt time
    err = encdec.GenerateKeysetFile(masterAEAD, ad, "keyset.json")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Wrote keyset.json")
}
```

This produces two artifacts:

| Artifact | Example | Where it goes |
|---|---|---|
| `MASTER_KEY` | `x7Kj2mN9pQ4rT1vW3yB5dF8hL0nS6uA2cE4gI7kM9o=` | K8s Secret (env var) |
| `keyset.json` | See below | ConfigMap or git |

**keyset.json** (metadata is visible, key material is encrypted):
```json
{
  "encryptedKeyset": "EYGbZSb6oz6C6IISsEZ9sVp2AvI...",
  "keysetInfo": {
    "primaryKeyId": 1546929177,
    "keyInfo": [
      {
        "typeUrl": "type.googleapis.com/google.crypto.tink.AesGcmKey",
        "status": "ENABLED",
        "keyId": 1546929177,
        "outputPrefixType": "TINK"
      }
    ]
  }
}
```

## 2. Application Configuration

### Config struct

```go
cfg := &crypto.CryptoConfig{
    MasterKey:     os.Getenv("MASTER_KEY"),          // KEK from env
    KeysetFile:    "/etc/myapp/keyset.json",         // DEK file path
    KekAd:         []byte("my-service keyset"),      // must match bootstrap
    HmacKey:       os.Getenv("HMAC_KEY"),            // for hashing
    HashAlgorithm: "siphash24",                      // or "sha256"
}

cu, err := crypto.NewCryptoUtil(cfg)
```

### Encrypt / Decrypt

```go
ctx := context.Background()

// Encrypt (returns base64 string)
ciphertext, err := cu.Encrypt(ctx, []byte("sensitive data"), []byte("context"))

// Decrypt
plaintext, err := cu.Decrypt(ctx, ciphertext, []byte("context"))

// Raw bytes (no base64 encoding)
cipherBytes, err := cu.EncryptRaw(ctx, []byte("sensitive data"), []byte("context"))
plainBytes, err  := cu.DecryptRaw(ctx, cipherBytes, []byte("context"))
```

### Hashing / Alias

```go
// Generate a deterministic hash (for lookups / indexing)
hash, err := cu.CreateAlias(ctx, []byte("user@example.com"))

// Constant-time comparison
match, err := cu.CompareHash(ctx, []byte("user@example.com"), hash)
```

## 3. Kubernetes Deployment

### 3a. Create the Secret (KEK)

```bash
# Generate the master key (or use the output from the bootstrap script)
MASTER_KEY=$(openssl rand -base64 32)

kubectl create secret generic crypto-kek \
  --from-literal=MASTER_KEY="${MASTER_KEY}" \
  --from-literal=HMAC_KEY="$(openssl rand -base64 32)" \
  -n my-namespace
```

### 3b. Create the ConfigMap (DEK keyset file)

```bash
kubectl create configmap crypto-keyset \
  --from-file=keyset.json=./keyset.json \
  -n my-namespace
```

### 3c. Deployment manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-service
spec:
  template:
    spec:
      containers:
        - name: my-service
          env:
            # KEK from Secret - only in memory, never on disk
            - name: MASTER_KEY
              valueFrom:
                secretKeyRef:
                  name: crypto-kek
                  key: MASTER_KEY
            - name: HMAC_KEY
              valueFrom:
                secretKeyRef:
                  name: crypto-kek
                  key: HMAC_KEY
          volumeMounts:
            # DEK keyset file from ConfigMap
            - name: keyset-volume
              mountPath: /etc/myapp
              readOnly: true
      volumes:
        - name: keyset-volume
          configMap:
            name: crypto-keyset
```

### What lives where

| Component | Storage | Sensitivity |
|---|---|---|
| Master Key (KEK) | K8s Secret -> env var | **HIGH** - never on disk, never in git |
| Keyset file (DEK) | ConfigMap / git | LOW - encrypted blob, useless without KEK |
| Associated data | App config | LOW - not secret, but must be consistent |
| HMAC key | K8s Secret -> env var | **HIGH** - same treatment as KEK |

## 4. Key Rotation

Key rotation adds a new encryption key without breaking existing ciphertext.
After rotation, new data is encrypted with the new key while old data can still
be decrypted with the old key.

### 4a. Run rotation

This can be a CI/CD job, a CronJob, or a manual step:

```go
masterAEAD, err := encdec.NewLocalAEAD(os.Getenv("MASTER_KEY"))
if err != nil {
    log.Fatal(err)
}

err = encdec.RotateKeysetFile(masterAEAD, []byte("my-service keyset"), "/path/to/keyset.json")
if err != nil {
    log.Fatal(err)
}
```

After rotation, `keyset.json` contains both keys:

```json
{
  "encryptedKeyset": "...",
  "keysetInfo": {
    "primaryKeyId": 3813839455,
    "keyInfo": [
      {"keyId": 881456094,  "status": "ENABLED"},
      {"keyId": 3813839455, "status": "ENABLED"}
    ]
  }
}
```

- `881456094` = old key, still ENABLED for decryption
- `3813839455` = new primary, used for all new encryptions

### 4b. Deploy the rotated keyset

```bash
# Update the ConfigMap
kubectl create configmap crypto-keyset \
  --from-file=keyset.json=./keyset.json \
  -n my-namespace \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart pods to pick up the new keyset
kubectl rollout restart deployment/my-service -n my-namespace
```

### 4c. Rotation as a Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: keyset-rotation
spec:
  schedule: "0 2 1 */3 *"  # Every 3 months at 2 AM on the 1st
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: rotator
              image: my-service:latest
              command: ["./my-service", "rotate-keyset"]
              env:
                - name: MASTER_KEY
                  valueFrom:
                    secretKeyRef:
                      name: crypto-kek
                      key: MASTER_KEY
              volumeMounts:
                - name: keyset-volume
                  mountPath: /etc/myapp
          volumes:
            - name: keyset-volume
              configMap:
                name: crypto-keyset
          restartPolicy: OnFailure
```

### Rotation does NOT require:
- Re-encrypting existing data
- Application downtime
- Changing the master key
- Database migrations

## 5. Migrating from caas-kms Mode

If you have an existing deployment using the old `caas-kms://` URI approach
(kmsUri string + keysetData string in config), migrate to JSON file + local KEK
without losing access to existing encrypted data.

The DEK (the key that actually encrypts your data) stays the same.
Only its wrapper changes: from cleartext-in-URI to encrypted-by-master-key.

### 5a. One-step migration: strings to JSON file

```go
package main

import (
    "fmt"
    "log"

    "github.com/achuala/go-svc-extn/pkg/crypto/encdec"
)

func main() {
    // Your current config values (from env, config file, etc.)
    oldCfg := &encdec.TinkConfiguration{
        KekUri:       "caas-kms://CILTuPkN...",  // your current kmsUri string
        KekUriPrefix: "caas-kms://",
        KeySetData:   "En0B3y4pgv8...",           // your current keysetData string
        KekAd:        []byte("caas kek"),         // your current associated data
    }

    // Generate a new master key — store this in K8s Secret
    newMasterKey, err := encdec.GenerateMasterKey()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("MASTER_KEY:", newMasterKey)

    // Migrate: decrypt DEK with old KEK, re-encrypt under new master key, write JSON
    newAd := []byte("my-service keyset")
    err = encdec.MigrateToKeysetFile(oldCfg, newMasterKey, newAd, "keyset.json")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Wrote keyset.json — same DEK, new wrapper")
}
```

This single call:
1. Reads your old `kmsUri` to extract the cleartext KEK
2. Decrypts the `keysetData` (your DEK) using that KEK
3. Re-encrypts the same DEK under the new master key
4. Writes it as an encrypted JSON file

### 5b. Update Kubernetes resources

```bash
# 1. Store the new master key
kubectl create secret generic crypto-kek \
  --from-literal=MASTER_KEY="<output from migration script>" \
  -n my-namespace

# 2. Store the keyset file
kubectl create configmap crypto-keyset \
  --from-file=keyset.json=./keyset.json \
  -n my-namespace

# 3. Update deployment (see section 3c for full manifest)
# 4. Restart
kubectl rollout restart deployment/my-service -n my-namespace
```

### 5c. Before and after

**Before (old config) — kmsUri + keysetData as strings:**
```yaml
kms_uri: "caas-kms://CILTuPkN..."       # cleartext KEK in URI (unsafe)
kms_uri_prefix: "caas-kms://"
keyset_data: "En0B3y4pgv8..."            # base64 binary blob
kek_ad: "caas kek"
```

**After (new config) — master key + JSON file:**
```yaml
master_key: ${MASTER_KEY}               # K8s Secret → env var (encrypted at rest)
keyset_file: "/etc/myapp/keyset.json"   # ConfigMap volume mount (encrypted by KEK)
kek_ad: "my-service keyset"
```

### What changed, what didn't

| | Before | After |
|---|---|---|
| **KEK** | Cleartext Tink keyset in URI string | AES-256 key in K8s Secret (env var) |
| **DEK** | Base64 binary blob in config string | Encrypted JSON file in ConfigMap |
| **DEK key material** | **Unchanged** | **Unchanged** |
| **Existing ciphertext** | **Decrypts normally** | **Decrypts normally** |
| **Key rotation** | Not possible without redeployment | `RotateKeysetFile()` + ConfigMap update |

## 6. Security Checklist

- [ ] `MASTER_KEY` is in a K8s Secret, not in config files, env files, or git
- [ ] `keyset.json` is encrypted (the `encryptedKeyset` field is opaque without the KEK)
- [ ] Associated data (`KekAd`) is consistent between bootstrap, rotation, and app config
- [ ] RBAC restricts access to the `crypto-kek` Secret to only the service account that needs it
- [ ] Key rotation is scheduled (quarterly recommended)
- [ ] Old `caas-kms://` URIs are removed from all config after migration
