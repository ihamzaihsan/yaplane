# Forum security extension

This extension keeps the Go-rendered pages, routes, SQLite schema, registration, login, discussions, comments, reactions, image uploads, and styling. Its changes concern transport security, session handling, request protection, and verification. The existing Font Awesome CDN stylesheet is replaced with a small local SVG sprite to meet the no-frontend-library rule. No frontend framework or additional third-party package is introduced.

## Local HTTPS setup

Run from the repository root with Go 1.22 or newer, CGO enabled, and a C compiler on PATH.

The original certificate in `HTTPS/` expired on October 24, 2025 and has no localhost subject alternative name. Its tracked files are left untouched. Generate a new self-signed certificate and key using the standard Go library:

```sh
go run ./cmd/certgen
```

This creates `.local/server.crt` and `.local/server.key`, valid for localhost, 127.0.0.1, and ::1 for one year. Existing files are never overwritten. To renew, choose new paths using `-cert` and `-key`. Additional hostnames can be provided with `-hosts localhost,127.0.0.1,example.test`.

Set the certificate paths before starting the server. In PowerShell:

```powershell
$env:TLS_CERT_FILE = '.local/server.crt'
$env:TLS_KEY_FILE = '.local/server.key'
go run .
```

In a POSIX shell:

```sh
TLS_CERT_FILE=.local/server.crt TLS_KEY_FILE=.local/server.key go run .
```

Open **https://localhost:8080**. A self-signed certificate still needs local trust or a browser exception. For public hosting, supply a certificate and key issued by a trusted CA.

`.env.example` documents the environment values; the server does not load `.env` files. `FORUM_ADDR` optionally changes the listener address, for example `127.0.0.1:8443`. Defaults retain `:8080`, `HTTPS/server.crt`, and `HTTPS/server.key` for compatibility. `.local/` and `.env` are excluded from both Git and the Docker build context.

For Docker, build with `docker build -f Docker/Dockerfile -t yaplane .`. Mount the absolute path to the generated `.local` directory read-only at `/certs`, and pass `TLS_CERT_FILE=/certs/server.crt` and `TLS_KEY_FILE=/certs/server.key` to the container. For example, in PowerShell:

```powershell
$certDirectory = (Resolve-Path .local).Path
docker run --rm -p 8080:8080 --mount "type=bind,source=$certDirectory,target=/certs,readonly" -e TLS_CERT_FILE=/certs/server.crt -e TLS_KEY_FILE=/certs/server.key yaplane
```

## Implemented controls

| Subject requirement | Implementation |
| --- | --- |
| HTTPS and TLS settings | Minimum TLS 1.2; explicit ECDHE/AES-GCM and ECDHE/ChaCha20 suites for TLS 1.2. TLS 1.3 uses Go's built-in suite selection. |
| Server timeouts | Header read: 5 seconds; request read and response write: 30 seconds; idle connections: 60 seconds; graceful shutdown: up to 10 seconds. Headers are limited to 1 MiB. |
| Rate limiting | One request per second per direct client IP, with a 20-request burst and a one-minute cooldown. HTTP 429 includes `Retry-After`. State is capped at 4,096 clients and idle entries are removed lazily. |
| Password protection | Existing bcrypt hashing and verification remain in place. Passwords are hashed, rather than reversibly encrypted. |
| Unique session identifiers | Cryptographically random version 4 UUIDs, generated with standard Go packages. Session state remains in SQLite. Randomness failures are returned as errors. |
| Cookie security | Host-only cookies with `Path=/`, `Secure`, `HttpOnly`, `SameSite=Lax`, and a 24-hour expiration. Logout clears the same cookie scope and deletes the server-side session. |
| Server-side expiration | Identity and profile lookups reject expired records. SQLite date conversion handles older records with timezone offsets. |
| One active session | Login replaces the user's previous session in a transaction. A single SQLite connection serializes writes and avoids concurrent replacement races. |
| Allowed packages | Standard Go, sqlite3, and bcrypt. The previous `golang.org/x/time` dependency is removed. |

Go's TLS 1.3 cipher suites are not configurable through `CipherSuites`; the explicit list applies to TLS 1.2. See the [Go TLS configuration documentation](https://pkg.go.dev/crypto/tls#Config).

State-changing requests must present a matching `Origin`, or a matching `Referer` when `Origin` is absent. Scheme, hostname, and port must match the request's origin; missing, opaque, and cross-site origins are rejected. This protects the existing HTML forms and `fetch` calls without changing the frontend. Scripts and command-line clients must also supply the forum's origin when sending POST requests. This follows the origin validation approach described in the [OWASP CSRF prevention guidance](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html#using-standard-headers-to-verify-origin).

Additional controls include method restrictions, 64 KiB limits for non-upload mutation bodies, expired-session checks on private routes, `no-store` on dynamic pages, anti-framing headers, and `nosniff`. Uploads are served using their detected image MIME type, even when a supplied filename ends in `.html`. Directory listings for uploads are disabled.

## Verification

Build and check the source with `go build ./...` and `go vet ./...`. The automated test files were removed during repository cleanup; their earlier versions remain in Git history. These commands do not verify browser workflows or live OAuth providers.

## Limits

This is a single-server school project. Rate limiting is in memory and uses the direct connection's client IP; reverse-proxy forwarding headers are deliberately not trusted. There is no distributed limiter or proxy-origin configuration. Origin checks assume the forum terminates HTTPS itself, which is also how its Docker setup runs.

Database encryption is an optional subject bonus and is not implemented. Existing SQLite data is preserved, but the database file must still be protected with appropriate filesystem access. Historical certificate keys committed in `HTTPS/` must not be used as public deployment credentials.

The image-upload review in [image-upload-review.md](image-upload-review.md) records the existing size-boundary and unused-file limitations separately. The main README is deferred at the owner's request.
