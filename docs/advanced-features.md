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
