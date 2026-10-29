# tb secret

`tb secret set|get|list|connect` is a skate-like key/value store for secrets.
It works offline as a local encrypted store. After `connect` it mirrors one
Bitwarden Secrets Manager project, and `set` writes through to it.

## Why Bitwarden Secrets Manager, and why a hand-written client

Bitwarden Secrets Manager's free plan covers personal use: unlimited secrets,
3 projects and 3 machine accounts. Its official `sdk-go` wraps a Rust library
through cgo, but GoReleaser builds with `CGO_ENABLED=0`. `bws` also has no brew
or winget package. So `internal/bitwarden` re-implements the small part of the
protocol `tb` needs, in pure Go. It was ported from
`bitwarden/sdk-internal`:

- `crates/bitwarden-core/src/auth/access_token.rs`
- `crates/bitwarden-crypto`
- `bitwarden_license/bitwarden-sm`

The test vectors in `token_test.go`, `crypto_test.go` and `client_test.go`
(the access token, the AES vector, and the fake-server login payload and org
key) are copied from that SDK. If Bitwarden changes the protocol, those tests
are where to start.

## Protocol notes

- **Access token**: `0.<access token id uuid>.<client secret>:<base64 16-byte seed>`.
  The base64 padding is optional.
- **Token key**: HMAC-SHA256 keyed with `"bitwarden-accesstoken"` over the
  seed, giving a PRK. Then HKDF-SHA256-expand with info `"sm-access-token"`
  to 64 bytes: `enc(32) || mac(32)`. This is `deriveShareableKey`.
- **Login**: POST `{identity}/connect/token`, form-encoded, with
  `grant_type=client_credentials`, `scope=api.secrets`,
  `client_id=<token id>` and `client_secret`.
  - The response's `encrypted_payload` decrypts with the token key to
    `{"encryptionKey": "<base64 64-byte org key>"}`.
  - The organization id is the `organization` claim of the JWT
    `access_token`.
- **Headers**: every request sends `Device-Type: 21` (the SDK device type);
  the server expects it.
- **EncString type 2**: `2.<b64 iv>|<b64 ciphertext>|<b64 mac>`, AES-256-CBC
  with PKCS7 padding, and MAC = HMAC-SHA256(mac key, iv || ciphertext).
  Project names, secret keys, values and notes are all EncStrings under the
  org key.
  - Only type 2 is supported. The newer COSE/XChaCha type 7 is not used for
    SM org data yet.
- **Endpoints** (relative to `{api}`):
  - `GET organizations/{org}/projects`
  - `GET projects/{id}/secrets`, which returns ids only, then
    `POST secrets/get-by-ids`
  - `POST organizations/{org}/secrets` with `projectIds`
  - `GET` and `PUT secrets/{id}`, with `valueChanged: true`
- **Regions**:
  - `us`: `identity.bitwarden.com` / `api.bitwarden.com`
  - `eu`: `.eu`
  - Self-hosted servers are not supported (only `--region`).
- **Update preserves the note**: `Update` GETs the secret first and
  re-encrypts its existing note, because PUT replaces the whole secret and
  would otherwise wipe notes edited in the web app.

## Storage

- **Store**: `os.UserConfigDir()/toolbelt/secrets.enc`.
  - Layout: `tbv1` magic, then a 12-byte nonce, then AES-256-GCM of the JSON
    `vault.Store`. The magic is also the AAD.
  - The JSON holds the entries (`value`, plus the Bitwarden `id` once synced)
    and the connection: token, region, org, project id and name.
  - The access token lives inside the encrypted file, so only one keyring
    item exists.
- **Data key**: 32 random bytes, base64, kept in the OS keyring
  (`zalando/go-keyring`: Keychain / Credential Manager / Secret Service) as
  service `toolbelt`, user `secret-store`.
- **Fallback**: if the keyring can't be reached, or refuses the write
  (headless Linux without a Secret Service), the key goes to
  `toolbelt/secret.key` with mode 0600, and a warning is logged when it is
  created. The fallback file is checked before generating a key, so a machine
  keeps using whichever place it started with.
- **Lock-out guard**: if no key is found but `secrets.enc` already exists,
  `LoadDataKey` returns `ErrKeyUnavailable` instead of minting a new key.
  Minting one would make the existing store undecryptable, for example when
  the keyring is just locked or `DBUS_SESSION_BUS_ADDRESS` is missing over
  ssh.
- **Atomic writes**: temp file in the same dir, chmod 0600, then rename. The
  dir is created 0700.

## Command behavior

- **`set KEY [VALUE]`**: when VALUE is omitted, a terminal gets a password
  prompt; otherwise stdin is read, with one trailing newline dropped.
  - Connected: the write goes to Bitwarden first (update when the entry has
    an id, create otherwise, and create again when the update 404s because
    the secret was deleted remotely). Only then is it saved locally. A failed
    push changes nothing locally.
- **`get KEY`**: reads only the local mirror, with no network, and prints
  the raw value plus a newline to stdout for piping.
- **`list`**: prints `KEY<TAB>value` sorted, masked with `secret.Redact`
  unless `--reveal`.
- **`connect`**:
  - The token comes from `--token`, then `$BWS_ACCESS_TOKEN`, then the stored
    connection, then a password prompt.
  - The project comes from `--project` (id or name, which must exist), then
    the stored project, then the only one, then a select prompt.
  - The region comes from `--region`, and otherwise the stored one, so
    rerunning `connect` with no flags refreshes the mirror.
- **Merge rules** (`vault.Merge`, pure):
  - Remote wins on a shared key.
  - A local entry that has no id is local-only: it is kept and offered for
    upload after a confirm. Declining, or running without a TTY, keeps it
    local only, and the next `connect` asks again.
  - A local entry that has an id but is missing remotely was deleted in
    Bitwarden, so it is dropped.
  - When switching to another project, local ids are forgotten first
    (`forgetRemoteIDs`), so the old project's secrets become local-only
    instead of being dropped.
- **Duplicate keys**: Bitwarden allows the same key twice in a project. The
  first one wins and a warning names the key.
- **Purge** (`vault.Purge`, run by `tb uninstall --purge`): removes
  `secrets.enc` and `secret.key`, then the `toolbelt` config dir only when it
  is left empty, then the keyring item. A missing keyring item is fine; an
  unreachable keyring is ignored when a `secret.key` existed (the key lived
  in the file, so the keyring never held it), otherwise it is an error. Files
  go before the keyring so a keyring failure never leaves a store whose key
  is gone. Bitwarden is never touched: purging only drops the local mirror.

## Known gaps

- No `delete`, no `disconnect`, no self-hosted URLs.
- Live login against bitwarden.com is not covered by CI. The client is tested
  against an `httptest` server that uses the SDK's real login payload.
- Each connected `set` logs in again, because the bearer token is not cached.
