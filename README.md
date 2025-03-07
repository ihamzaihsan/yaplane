# Yaplane Community: Full Stack Discussion Forum

A full-stack discussion forum built with **Go, SQLite, and server-rendered HTML**, bringing categorized discussions, image uploads, account management, and community moderation into a responsive interface.

The project demonstrates HTTP handler development, session authentication, relational data modeling, authorization, and transactional persistence. One Go server serves the pages, static assets, and forum endpoints.

## Key features

- **Discussions:** posts with multiple topics, optional image attachments, comments, and likes/dislikes on posts and comments.
- **Personal activity:** created posts, liked posts, comment history, and reaction history; authors can edit their own posts/comments and delete their own content.
- **Notifications:** persistent alerts for reactions and comments on your posts, unread counts, and a mark-all-as-read action.
- **Accounts:** email/password registration and login, optional Google/GitHub sign-in, profiles, bcrypt password hashing, and expiring sessions.
- **Moderation:** moderator requests, staff approval queues, moderator reports, administrator replies, role management, and managed topics.
- **Request protection:** HTTPS, Secure/HttpOnly/SameSite cookies, same-origin checks for mutations, rate limiting, and server timeouts.
- **Interface:** responsive layouts, topic filters, local SVG icons, labelled forms, and keyboard focus indicators.

## Technology stack

| Layer | Technology |
| --- | --- |
| Backend | Go 1.22+, `net/http`, `html/template`, `database/sql` |
| Database | SQLite through `mattn/go-sqlite3` with CGO |
| Authentication | bcrypt, UUID session identifiers, Google/GitHub OAuth |
| Frontend | HTML, CSS, and plain JavaScript |
| Local environment | Docker and Docker Compose; Docker builds with Go 1.27.1 |

The frontend needs no framework, npm install, or build step. Pages use navigation and form submissions; JavaScript handles reaction requests and checks notification counts every 15 seconds while the inbox is visible. SQLite transactions enforce permissions and content updates, and database triggers persist notifications with the corresponding reactions/comments. A single database connection serializes writes.

## Run locally

Install Docker with Compose support and start Docker. Run these commands from the repository root. For a populated demo, use **Start with demo content** instead of this first startup:

```sh
docker compose up --build -d
```

Open **[https://localhost:8080](https://localhost:8080)** and register an account. The image generates a self-signed localhost certificate; your browser will ask you to accept it for local use.

The application creates an empty database and five default topics automatically. Separate Docker volumes preserve the database and uploaded images across restarts.

```sh
docker compose down
```

This stops the app while keeping its data. Optional settings are listed in [.env.example](.env.example). Docker Compose reads settings from `.env`; native Go runs read shell environment variables.

### Native Go

Install Go 1.22 or newer and a C compiler with CGO enabled. From the repository root, generate the development certificate once and start the forum:

```sh
go run ./cmd/certgen
go run .
```

Reuse existing certificates on later starts. The generator refuses to overwrite files; use new `-cert` and `-key` paths when renewing and set `TLS_CERT_FILE` and `TLS_KEY_FILE` accordingly.

The native server uses **https://localhost:8080** and stores its database at `data/forum.db`. `DATABASE_PATH` changes database storage; `FORUM_ADDR` changes the native listener. Database files, generated certificates, and uploads are excluded from Git.

## Start with demo content

The demo uses its own Compose project and volumes, keeping it separate from standard data. Stop the standard forum first with `docker compose down` to free port 8080, then run:

```sh
docker compose -p yaplane-community-demo build
docker compose -p yaplane-community-demo run --rm forum /app/seed-demo
docker compose -p yaplane-community-demo up -d
```

**Git Bash on Windows:** use this version to prevent Git Bash from converting `/app/seed-demo` into a Windows path:

```bash
docker compose -p yaplane-community-demo build
MSYS_NO_PATHCONV=1 docker compose -p yaplane-community-demo run --rm forum /app/seed-demo
docker compose -p yaplane-community-demo up -d
```

Open **https://localhost:8080**. The seed creates **5 fictional accounts, 15 posts, 30 replies, and 90 reactions**. Post reactions and comments also populate the notification inboxes.

| Username | Sign-in email |
| --- | --- |
| alex_demo | alex_demo@example.com |
| mia_demo | mia_demo@example.com |
| sam_demo | sam_demo@example.com |
| noor_demo | noor_demo@example.com |
| leo_demo | leo_demo@example.com |

**Password for all demo accounts:** `DemoReview!2026`. Sign in using the email address. These are ordinary member accounts.

Browse topics, react to discussions, add a comment, then explore **Activity** and **Notifications**. Edit your own content to try the ownership controls. The homepage shows the latest ten posts; older discussions remain accessible through topic filters and personal activity.

Seed before the first demo startup. The command refuses any existing database file and never overwrites it. On later starts, use `docker compose -p yaplane-community-demo up -d` without seeding again. Stop the demo with `docker compose -p yaplane-community-demo down`.

For native Go, seed a new file and select it in PowerShell before starting:

```powershell
go run ./cmd/seed-demo -database data/demo.db
$env:DATABASE_PATH = 'data/demo.db'
go run .
```

If the demo file already exists, skip the seed command. To return to standard storage after stopping the server, run `Remove-Item Env:DATABASE_PATH` and `go run .`. See [running.md](docs/running.md) for running both Docker versions together on separate ports.

## Authentication and moderation setup

Local accounts work without external credentials. Enable Google/GitHub sign-in using your own OAuth applications and matching HTTPS callback URLs, as described in [authentication.md](docs/authentication.md).

Register a private account, then assign the first administrator through the local command:

```sh
go run ./cmd/admin -email your-account@example.com
```

For the standard Docker project, use `docker compose exec forum /app/admin -email your-account@example.com`. Add `-p yaplane-community-demo` when targeting the demo project. The command selects `DATABASE_PATH` or accepts an explicit `-db` path.

Members publish immediately by default. Set `FORUM_PREMODERATE=true` before startup to require staff approval for new member posts/comments. Pending content is visible to its author and staff; role permissions are checked server-side. See [moderation.md](docs/moderation.md) for the complete workflow.

## Verification and technical limits

```sh
go build ./...
go vet ./...
docker compose config --quiet
```

Local verification covered clean and seeded HTTPS startup, demo login, discussion/topic pages, activity, notifications, database integrity, refusal to overwrite an existing seed file, Docker storage isolation, and persistence after restart. Automated test files are not included in the current tree. Live Google/GitHub provider sign-in and public deployment were not verified.

The upload limit currently applies to the entire 20 MiB multipart request. Rejected submissions can leave unused attachments, and deleting content does not remove its physical image file. See [image-upload-review.md](docs/image-upload-review.md) for these limitations. Development certificates are self-signed; public hosting requires trusted certificates. Rate limiting is in memory and uses the direct client IP.

Further implementation details are in [security.md](docs/security.md) and [advanced-features.md](docs/advanced-features.md).
