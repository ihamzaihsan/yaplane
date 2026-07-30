import { mkdir, readFile, copyFile, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

const source = new URL('./', import.meta.url);
const output = new URL('./dist/', source);
const go = await readFile(new URL('../database/demo.go', source), 'utf8');
// Read the existing fictional fixture rather than maintaining a second content list.
const literal = '"(?:[^"\\\\]|\\\\.)*"';
const row = new RegExp(`\\{(${literal}), (${literal}), (${literal}), \\[2\\]string\\{(${literal}), (${literal})\\}\\}`, 'g');
const fixtures = [...go.matchAll(row)].map(match => match.slice(1).map(value => JSON.parse(value)));
const memberList = go.match(/members := \[\]string\{([^}]+)\}/);
if (!memberList || fixtures.length !== 15) throw new Error('Demo fixture format changed; update the preview exporter.');
const names = JSON.parse(`[${memberList[1]}]`);
const now = Date.now();
const state = {
    version: 1, viewer: 1, premoderate: false,
    users: names.map((name, i) => ({ id: i + 1, name, role: 'user' })),
    topics: ['science', 'technology', 'art', 'sport', 'games'],
    posts: [], comments: [], reactions: [], notifications: [],
};
function notify(userId, actorId, postId, kind, commentId = null) {
    if (userId !== actorId) state.notifications.push({ id: state.notifications.length + 1, userId, actorId, postId, kind, commentId, read: false });
}
fixtures.forEach(([topic, title, content, ...replies], i) => {
    const author = i % names.length;
    const post = { id: i + 1, userId: author + 1, title, content, topics: [topic], status: 'approved', image: '', createdAt: now - (fixtures.length - i) * 12 * 3600000 };
    state.posts.push(post);
    replies.forEach((content, j) => {
        const comment = { id: state.comments.length + 1, postId: post.id, userId: (author + j + 1) % names.length + 1, content, status: 'approved', createdAt: post.createdAt + (j + 1) * 3600000 };
        state.comments.push(comment);
        state.reactions.push({ userId: (author + j + 3) % names.length + 1, kind: 'comment', targetId: comment.id, like: true });
        notify(post.userId, comment.userId, post.id, 'commented', comment.id);
    });
    for (let j = 1; j < names.length; j++) {
        const userId = (author + j) % names.length + 1;
        const like = j <= 1 + i % 4;
        state.reactions.push({ userId, kind: 'post', targetId: post.id, like });
        notify(post.userId, userId, post.id, like ? 'liked' : 'disliked');
    }
});
await mkdir(new URL('static/', output), { recursive: true });
for (const file of ['index.html', 'app.js', 'store.js', 'preview.css']) {
    await copyFile(new URL(file, source), new URL(file, output));
}
for (const file of ['style.css', 'theme.js', 'icons.svg', 'favicon.svg']) {
    await copyFile(new URL(`../static/${file}`, source), new URL(`static/${file}`, output));
}
await writeFile(new URL('seed.json', output), JSON.stringify(state));
console.log(`Static preview built at ${fileURLToPath(output)} (5 accounts, 15 discussions, 30 comments, 90 reactions).`);
