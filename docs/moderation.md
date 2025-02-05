# Forum moderation extension

The forum has four access levels: guests, members (`user`), moderators, and administrators. Registration always creates a member account, including Google/GitHub accounts. A registration checkbox can submit a moderator request; existing members can request the role from **Profile → community tools** or `/moderation`.

## First administrator

Start the forum once to update the schema, then register the account you want to use. From the repository directory, assign that existing account through local database access:

```powershell
go run ./cmd/admin -email 'your-account@example.com'
```

Use `-db` if the database has another path. The command refuses a missing database or an unknown email. It does not register an account or set a password. No HTTP endpoint can grant the administrator role. Existing sessions pick up the new role immediately; sign in and open `/moderation`.

For Docker, use the command against the running container's database:

```sh
docker exec <container-name> go run ./cmd/admin -email your-account@example.com
```

The current Dockerfile retains Go, so this command is available inside the container. Use the container's database when the application runs there; promoting an account in a separate host database does not update the container.

## Moderation workflow

| Role | Permissions |
| --- | --- |
| Guest | View published discussions, comments, and reaction counts. Mutation endpoints require a valid session. |
| Member | Existing posting, commenting, reactions, profile, and filters; delete own posts; request moderator access and see the administrator's response. |
| Moderator | Member permissions; review pending content; delete any post; report posts as irrelevant, obscene, illegal, or insulting; see replies to own reports. |
| Administrator | Moderator permissions; accept/decline role requests with a reply; promote/demote moderators; answer reports; delete comments; create/delete topics. |

The moderation page includes pending content, requests, reports and replies, plus role/topic management for administrators. Moderators can report a post from its discussion page using **Report to administrator**. Admin replies appear in the reporting moderator's community tools page. Reports retain the post title and replies after the post is deleted. Demotion removes staff permissions immediately, while the member retains access to their own previous reports and replies.

An unanswered report from the same moderator on the same post cannot be submitted twice. An administrator can respond once; the moderator may submit a new report after that response. Pending role requests cannot be duplicated. Declined requests can be submitted again. An administrator can also promote a member directly; pending requests are marked accepted.

Topics appear dynamically in the homepage filters and create-post form. A topic used by any post cannot be deleted, preserving existing discussions and their associations. Delete the posts using it first if its removal is necessary. Deleted default topics stay deleted after restart. Topic names must contain 1–80 bytes and cannot contain commas, because the existing category display uses comma-separated query results.

## Approval before publication

Immediate publication remains the default, preserving existing functionality. To require approval for new member posts and comments, set this before starting the server:

```powershell
$env:FORUM_PREMODERATE = 'true'
go run .
```

For Docker, pass `-e FORUM_PREMODERATE=true`. Moderators and administrators publish directly. Existing content remains approved when the schema is updated; enabling review does not retroactively hide it.

Pending posts appear in the author's My Posts page and the staff approval queue. The author and staff can preview the post and its image. Guests and other members cannot retrieve pending posts or their attached images through the homepage, filters, direct URLs, or reaction endpoints. Pending comments are visible only to their author and staff until approved. Comments and reactions cannot be added to a pending post. Reactions cannot be added to a pending comment.

Staff approve content from `/moderation`. Rejecting content deletes it. Turning review off affects new submissions; already pending submissions still need approval or rejection. Review is manual; there is no automatic profanity classifier or keyword censorship.
