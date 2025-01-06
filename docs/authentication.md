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
