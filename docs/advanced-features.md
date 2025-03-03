# Forum advanced features

The extension adds a private activity page, persistent notifications, and author-controlled editing/removal. It uses the existing Go, SQLite, session system and server-rendered pages. No dependency or frontend library was added.

## Activity and managing content

Open **My activity** from the homepage sidebar, or **View and manage my activity** from your profile. `/activity` shows:

- Your discussions, including submissions awaiting approval.
- Your comments with their content and the discussion they belong to.
- Your current likes/dislikes on posts and comments, with links to their context.

The page includes edit/delete controls for your own posts and comments. Discussion and comment pages also expose those controls to their authors. Deleted items disappear from activity; changing a reaction replaces the current choice rather than creating an immutable activity log.

Post editing changes the title, content and topics while retaining the existing image, author, ID and creation time. Comment editing changes its content. At least one valid topic is required for a post; unknown or duplicate topic IDs are rejected without changing the original data. Titles are limited to 200 bytes and content to 20,000 bytes. Invalid edits return HTTP 400 with a message and retain the submitted text in the form.

Only authors can edit, including when another user is an administrator. Authors can delete their own comments; administrators retain their existing right to delete anyone's comments. Existing moderator/admin post-deletion permissions remain in place. Ownership and roles are checked in SQLite transactions, independently of which controls appear in the browser. Guests cannot access private activity, notifications or editing endpoints.

## Notifications

Open **Notifications** from the sidebar or profile. The profile shows an unread count. The inbox displays the latest 100 notifications, identifies who reacted/commented and links to the affected discussion. **Mark all as read** saves read state for your account, including older notifications outside the displayed window.

Post owners receive notifications when another user likes, dislikes or comments on their posts. Self-reactions/comments do not create notifications. Repeating the same reaction does not create another alert; switching like/dislike does. Reactions on comments appear in activity, but do not send additional notifications because the subject requires notifications for reactions on posts.

Notifications are created in the same SQLite write as the reaction/comment through triggers. A failed notification write rolls back the associated change. Reaction writes are serialized in transactions, preventing concurrent requests from creating duplicate reactions or alerts. Inbox entries and read state persist across sessions and server restarts. Existing reactions/comments are not backfilled into notifications.

While the inbox is open and visible, a short plain-JavaScript check polls `/notifications/count` every 15 seconds. A new event refreshes the inbox; unread-count changes update the displayed count. This is polling, not a WebSocket connection or browser push notification. Without JavaScript, the page still works with ordinary refreshes. Other pages display their server-rendered state until refreshed.

## Moderation integration

With `FORUM_PREMODERATE=true`, editing a member's approved post/comment returns it to the pending state. Already pending content stays pending even if review is later disabled. Staff editing their own approved content retain immediate publication. Posts and their attached images stay private during review; pending comments are hidden from other members and guests.

Pending comments do not notify the post owner until approved. If an approved comment is edited back into review, its comment notifications are removed until reapproval. Publish the parent post before approving its pending comments; otherwise approval returns HTTP 409. This prevents a comment from being marked public while its discussion is still awaiting review.

Activity retains your own comment text even if its parent post becomes private. In that case, an unauthorized viewer sees “Discussion awaiting review” instead of the pending post's title or link. Reactions referencing pending posts/comments are hidden until publication resumes.

Removing posts/comments cascades to their corresponding notifications, avoiding links to deleted content. Physical uploaded-file cleanup remains the existing limitation documented in [image-upload-review.md](image-upload-review.md). The earlier security, authentication and moderation setup still applies; no new credentials or environment settings are needed.

## Verification

Build and check the source with `go build ./...` and `go vet ./...`. The automated test files were removed during repository cleanup; their earlier versions remain in Git history. These commands do not verify browser workflows or live OAuth providers.
