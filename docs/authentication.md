# Forum authentication extension

Google and GitHub can register and sign in through the existing Go server. Email/password registration and login remain available. Provider accounts use the same session cookies and permissions as local accounts: posts, comments, reactions, profile, filters, and ownership checks.

No dependency was added. OAuth authorization-code exchanges, provider API requests, cookies, state signatures, and PKCE use the Go standard library. The two new buttons use server-rendered HTML and existing styling; no browser SDK is required.

## Configuration

Set up local HTTPS first using [security.md](security.md). Then register your own applications:

- **Google:** create a Web application OAuth client in Google Cloud, configure the consent screen, and add `https://localhost:8080/auth/google/callback` as an authorized redirect URI. If the application is in testing mode, add the accounts you will use as test users. See [Google's server authentication documentation](https://developers.google.com/identity/openid-connect/openid-connect).
- **GitHub:** create an OAuth App in your developer settings, set the homepage to `https://localhost:8080`, and the authorization callback to `https://localhost:8080/auth/github/callback`. See [GitHub's OAuth authorization documentation](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps).

Set these environment variables in your local shell before starting the forum:

```powershell
$env:OAUTH_BASE_URL = 'https://localhost:8080'
$env:GOOGLE_CLIENT_ID = '<your Google client ID>'
$env:GOOGLE_CLIENT_SECRET = '<your Google client secret>'
$env:GITHUB_CLIENT_ID = '<your GitHub client ID>'
$env:GITHUB_CLIENT_SECRET = '<your GitHub client secret>'
$env:TLS_CERT_FILE = '.local/server.crt'
$env:TLS_KEY_FILE = '.local/server.key'
go run .
```

Keep real secrets outside tracked files. `.env.example` is a reference; the application does not load `.env` automatically. For Docker, pass the same variables with `--env-file` or `-e` alongside the certificate mount described in the security guide.

Use the exact configured origin in the browser. If you change the host or external port, update `OAUTH_BASE_URL` and both registered callback URLs. The base URL must be an HTTPS origin, without credentials, a query, a fragment, or a path. Restart after changing configuration. Leaving both credentials empty disables that provider and hides its button. Incomplete credentials or an invalid origin disable external authentication and produce a startup log message; local login continues working.

## Account and session behavior

The additive `oauth_identities` table maps a provider's stable subject/user ID to a forum user. Existing tables and records are preserved. First sign-in creates an account with a provider-derived display name and a short random suffix. Later sign-ins find that same account even if the provider email changes; they retain the original forum email and all posts.

Google must return a verified email. GitHub's primary email must be verified, retrieved using the `user:email` permission, including when it is private. Accounts without such an email receive an error. Each login replaces the user's previous forum session, with the existing 24-hour expiration and logout behavior.

Matching emails never automatically link accounts. If an email already belongs to a local account or a different provider identity, sign-in shows an error directing the user to the original method. Explicit account linking, password setup/reset for provider accounts, and email synchronization are outside this extension. Provider-only accounts have an empty password hash, which bcrypt rejects; they sign in through their provider.

The browser is bound to the flow using a signed, provider-specific, Secure/HttpOnly/SameSite=Lax cookie that expires after ten minutes. The authorization request uses random state and an S256 PKCE challenge; callbacks check state, signature, provider, and issuance time before any exchange. The authorization code is exchanged server-side. Access tokens are used only to fetch identity and are never stored in SQLite or sent to the browser. Provider requests have timeouts, bounded response reads, and do not follow redirects. Restarting the forum invalidates pending flows because the signing key is held in memory.

Local registration now also rejects malformed emails and passwords exceeding bcrypt's 72-byte limit. Duplicate usernames/emails return HTTP 409 with an error message; missing fields and mismatched passwords return HTTP 400. Existing password handling and bcrypt hashing are retained.
