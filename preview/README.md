# Yaplane interactive preview

A browser-only companion to the Go + SQLite forum. It reuses the application's styles, icons, theme toggle, and fictional discussions from `database/demo.go`. The original Go application remains available in this branch.

## Run locally

Install Node.js 22 or newer. No npm packages or environment variables are required. From the repository root:

```sh
node preview/build.mjs
python -m http.server 4173 --bind 127.0.0.1 --directory preview/dist
```

Open `http://localhost:4173`. Python 3 is used only as a local file server; any static HTTP server can serve `preview/dist`. Opening the HTML directly with `file://` will not load its modules and fixture correctly.

The build copies only the preview and required frontend assets into `preview/dist`, exports the existing fictional fixture, and excludes backend source, certificates, database files, and private configuration. Generated output is ignored by Git.

## Available interactions

- Browse 15 fictional discussions, 30 comments, and 90 seeded reactions; filter discussions by topic.
- Switch between five fictional personas or explore as a guest, without credentials.
- Create, edit, and delete local discussions/comments; react to published content.
- Attach PNG, JPEG, GIF, or WebP images up to 1 MiB. Attachments are decoded before saving.
- Explore personal activity, notifications, unread counts, and read state.
- Change a persona's simulated role and try the staff approval queue, including optional member premoderation.
- Switch themes and reset the fictional forum.

Routes use URL fragments (for example, `/#/posts/15`), so navigation, bookmarks, browser history, and refreshes work on a static host without server rewrites.

## Storage and scope

Preview data is saved under `yaplanePreview.v1` in the current browser's `localStorage`. It is separate from the original SQLite database and other visitors' browsers. Tabs on the same origin synchronize saved changes. Browser storage is finite; failed writes leave the previous data unchanged and display an error. If storage is blocked, the preview works in memory and warns that reloads discard changes. Invalid saved data restores the fictional fixture. Reset clears preview data but preserves the theme preference.

Account switching and role permissions simulate application flows. They provide no real authentication or security boundary. Google/GitHub OAuth, registration, moderator requests/reports, topic administration, backend rate limiting, and server persistence are available only in the Go application. Notifications reflect local preview actions; there is no server polling, email, or push delivery. Changes do not reach GitHub or a shared community.

Local and public-deployment verification passed in Chrome: topic filtering, fragment-route refreshes, input validation, escaped user content, image attachments, content editing/deletion, reaction replacement, notification read state, role switching and moderation, persistence, reset, theme persistence, and layouts from 320 to 1440 pixels wide. Additional checks covered cross-tab synchronization, visitor isolation, blocked/corrupt browser storage, invalid/oversized attachments, and failed-write rollback. No console errors or failed network requests occurred in those workflows. Other browser engines were not verified.

Automated verification scripts are temporary and are not retained in the repository. The preview can be verified through the browser workflows above; `node --check preview/app.js` and `node --check preview/store.js` check JavaScript syntax.
