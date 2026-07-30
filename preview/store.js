// This is a browser simulation, not an authentication or authorization boundary.
const STORAGE_KEY = 'yaplanePreview.v1';
const roles = ['user', 'moderator', 'admin'];
const statuses = ['approved', 'pending'];
const text = (value, max) => typeof value === 'string' && value.length <= max;
const positive = value => Number.isSafeInteger(value) && value > 0;
export const safeImage = value => value === '' || (text(value, 1500000) && /^data:image\/(png|jpeg|gif|webp);base64,[A-Za-z0-9+/=]+$/.test(value));

// Treat localStorage as untrusted input: stale or damaged data must not break pages.
function validState(s) {
    if (!s || s.version !== 1 || typeof s.premoderate !== 'boolean') return false;
    for (const [name, limit] of [['users', 5], ['topics', 50], ['posts', 500], ['comments', 2000], ['reactions', 10000], ['notifications', 1000]]) {
        if (!Array.isArray(s[name]) || s[name].length > limit) return false;
    }
    if (s.users.length !== 5 || !s.users.every(u => u && positive(u.id) && text(u.name, 50) && roles.includes(u.role))) return false;
    const unique = list => new Set(list.map(item => item.id)).size === list.length;
    if (![s.users, s.posts, s.comments, s.notifications].every(unique)) return false;
    const users = new Set(s.users.map(u => u.id));
    if (s.viewer !== null && !users.has(s.viewer)) return false;
    if (!s.topics.length || !s.topics.every(t => text(t, 80) && t.trim()) || new Set(s.topics).size !== s.topics.length) return false;
    if (!s.posts.every(p => p && positive(p.id) && users.has(p.userId) && text(p.title, 200) && p.title.trim() && text(p.content, 20000) && p.content.trim() && statuses.includes(p.status) && safeImage(p.image) && Number.isFinite(p.createdAt) && Array.isArray(p.topics) && p.topics.length > 0 && p.topics.every(t => s.topics.includes(t)) && new Set(p.topics).size === p.topics.length)) return false;
    const posts = new Set(s.posts.map(p => p.id));
    if (!s.comments.every(c => c && positive(c.id) && users.has(c.userId) && posts.has(c.postId) && text(c.content, 20000) && c.content.trim() && statuses.includes(c.status) && Number.isFinite(c.createdAt))) return false;
    const comments = new Set(s.comments.map(c => c.id));
    if (!s.reactions.every(r => r && users.has(r.userId) && typeof r.like === 'boolean' && ((r.kind === 'post' && posts.has(r.targetId)) || (r.kind === 'comment' && comments.has(r.targetId))))) return false;
    if (new Set(s.reactions.map(r => `${r.userId}:${r.kind}:${r.targetId}`)).size !== s.reactions.length) return false;
    return s.notifications.every(n => n && positive(n.id) && users.has(n.userId) && users.has(n.actorId) && posts.has(n.postId) && (n.commentId === null || s.comments.some(c => c.id === n.commentId && c.postId === n.postId)) && ['liked', 'disliked', 'commented'].includes(n.kind) && typeof n.read === 'boolean');
}

export class PreviewStore {
    constructor(seed) {
        if (!validState(seed)) throw new Error('The fictional demo data could not be loaded.');
        this.seed = structuredClone(seed);
        this.state = structuredClone(seed);
        this.warning = '';
        this.persistent = true;
        try {
            const saved = localStorage.getItem(STORAGE_KEY);
            if (saved) {
                try {
                    const parsed = JSON.parse(saved);
                    if (!validState(parsed)) throw new Error('Invalid saved data');
                    this.state = parsed;
                } catch {
                    localStorage.removeItem(STORAGE_KEY);
                    this.warning = 'Saved preview data was unreadable. The fictional community has been restored.';
                }
            }
        } catch {
            this.persistent = false;
            this.warning = 'Browser storage is unavailable. You can explore, but changes will be lost when this page reloads.';
        }
    }

    get user() { return this.state.users.find(u => u.id === this.state.viewer); }
    isStaff(user = this.user) { return user?.role === 'moderator' || user?.role === 'admin'; }
    canView(post) { return post && (post.status === 'approved' || post.userId === this.user?.id || this.isStaff()); }
    canViewComment(comment) { return this.canView(this.state.posts.find(p => p.id === comment.postId)) && (comment.status === 'approved' || comment.userId === this.user?.id || this.isStaff()); }
    name(id) { return this.state.users.find(u => u.id === id)?.name ?? 'Demo member'; }
    counts(kind, targetId) {
        const reactions = this.state.reactions.filter(r => r.kind === kind && r.targetId === targetId);
        return { likes: reactions.filter(r => r.like).length, dislikes: reactions.filter(r => !r.like).length };
    }
    unread() { return this.state.notifications.filter(n => n.userId === this.user?.id && !n.read && this.canView(this.state.posts.find(p => p.id === n.postId))).length; }

    change(update) {
        const next = structuredClone(this.state);
        update(next);
        if (!validState(next)) throw new Error('This preview has reached its data limit. Reset the demo to keep exploring.');
        if (this.persistent) {
            try { localStorage.setItem(STORAGE_KEY, JSON.stringify(next)); }
            catch { throw new Error('Browser storage is full or unavailable. Remove an attachment or reset the demo. Your previous data is unchanged.'); }
        }
        this.state = next;
    }

    requireUser() {
        if (!this.user) throw new Error('Choose a fictional account above to try this action.');
        return this.user;
    }
    selectViewer(id) {
        if (id !== null && !this.state.users.some(u => u.id === id)) throw new Error('Choose an available demo account.');
        this.change(s => { s.viewer = id; });
    }
    setRole(role) {
        const user = this.requireUser();
        if (!roles.includes(role)) throw new Error('Choose a valid preview role.');
        this.change(s => { s.users.find(u => u.id === user.id).role = role; });
    }
    setPremoderate(enabled) {
        if (!this.isStaff()) throw new Error('Explore as a moderator or administrator to change review settings.');
        this.change(s => { s.premoderate = enabled; });
    }
    reset() {
        if (this.persistent) {
            try { localStorage.removeItem(STORAGE_KEY); }
            catch { throw new Error('Your browser could not clear saved preview data.'); }
        }
        this.state = structuredClone(this.seed);
    }
    reload() {
        if (!this.persistent) return;
        const saved = localStorage.getItem(STORAGE_KEY);
        const next = saved ? JSON.parse(saved) : structuredClone(this.seed);
        if (!validState(next)) throw new Error('Saved preview data changed unexpectedly. Reset the demo to restore it.');
        this.state = next;
    }

    savePost({ id, title, content, topics, image }) {
        const user = this.requireUser();
        title = title.trim(); content = content.trim();
        if (!title || title.length > 200 || !content || content.length > 20000) throw new Error('Enter a title (up to 200 characters) and content (up to 20,000 characters).');
        if (!topics.length || topics.some(t => !this.state.topics.includes(t)) || new Set(topics).size !== topics.length) throw new Error('Choose at least one available topic.');
        if (!safeImage(image)) throw new Error('Choose a PNG, JPEG, GIF, or WebP image up to 1 MiB.');
        let result;
        this.change(s => {
            const existing = id ? s.posts.find(p => p.id === id) : null;
            if (id && (!existing || existing.userId !== user.id)) throw new Error('Only the author can edit this discussion.');
            const status = existing?.status === 'pending' || (s.premoderate && user.role === 'user') ? 'pending' : 'approved';
            if (existing) {
                Object.assign(existing, { title, content, topics, image, status });
                result = existing.id;
            } else {
                result = nextId(s.posts);
                s.posts.push({ id: result, userId: user.id, title, content, topics, image, status, createdAt: Date.now() });
            }
        });
        return result;
    }

    saveComment(postId, content, id = null) {
        const user = this.requireUser();
        content = content.trim();
        if (!content || content.length > 20000) throw new Error('Enter a comment up to 20,000 characters.');
        this.change(s => {
            const post = s.posts.find(p => p.id === postId);
            const existing = id ? s.comments.find(c => c.id === id && c.postId === postId) : null;
            if (!post || (!existing && post.status !== 'approved')) throw new Error('Comments can be added to published discussions.');
            if (id && (!existing || existing.userId !== user.id)) throw new Error('Only the author can edit this comment.');
            const status = existing?.status === 'pending' || (s.premoderate && user.role === 'user') ? 'pending' : 'approved';
            if (existing) {
                Object.assign(existing, { content, status });
                if (status === 'pending') s.notifications = s.notifications.filter(n => n.commentId !== existing.id);
            } else {
                const comment = { id: nextId(s.comments), postId, userId: user.id, content, status, createdAt: Date.now() };
                s.comments.push(comment);
                if (status === 'approved') notify(s, post.userId, user.id, postId, 'commented', comment.id);
            }
        });
    }

    react(kind, id, like) {
        const user = this.requireUser();
        if (!['post', 'comment'].includes(kind)) throw new Error('Invalid reaction.');
        this.change(s => {
            const item = (kind === 'post' ? s.posts : s.comments).find(item => item.id === id);
            const post = kind === 'post' ? item : s.posts.find(p => p.id === item?.postId);
            if (!item || item.status !== 'approved' || post?.status !== 'approved') throw new Error('Reactions are available on published content.');
            const existing = s.reactions.find(r => r.userId === user.id && r.kind === kind && r.targetId === id);
            if (existing?.like === like) return;
            if (existing) existing.like = like;
            else s.reactions.push({ userId: user.id, kind, targetId: id, like });
            if (kind === 'post') notify(s, post.userId, user.id, id, like ? 'liked' : 'disliked');
        });
    }

    deleteContent(kind, id, review = false) {
        const user = this.requireUser();
        this.change(s => {
            const item = (kind === 'post' ? s.posts : s.comments).find(item => item.id === id);
            if (!item) throw new Error('This item has already been removed.');
            if (review && (!this.isStaff() || item.status !== 'pending')) throw new Error('Only staff can reject pending content.');
            if (!review && item.userId !== user.id && !(kind === 'post' ? this.isStaff() : user.role === 'admin')) throw new Error('Your preview role cannot delete this item.');
            const commentIds = kind === 'post' ? s.comments.filter(c => c.postId === id).map(c => c.id) : [id];
            if (kind === 'post') s.posts = s.posts.filter(p => p.id !== id);
            s.comments = s.comments.filter(c => !commentIds.includes(c.id));
            s.reactions = s.reactions.filter(r => !(r.kind === 'post' && kind === 'post' && r.targetId === id) && !(r.kind === 'comment' && commentIds.includes(r.targetId)));
            s.notifications = s.notifications.filter(n => !(kind === 'post' && n.postId === id) && !commentIds.includes(n.commentId));
        });
    }

    approve(kind, id) {
        if (!this.isStaff()) throw new Error('Choose a moderator or administrator preview role to approve content.');
        this.change(s => {
            const item = (kind === 'post' ? s.posts : s.comments).find(item => item.id === id);
            if (!item || item.status !== 'pending') throw new Error('This item is no longer awaiting review.');
            if (kind === 'comment') {
                const post = s.posts.find(p => p.id === item.postId);
                if (post.status !== 'approved') throw new Error('Approve the parent discussion first.');
                notify(s, post.userId, item.userId, post.id, 'commented', item.id);
            }
            item.status = 'approved';
        });
    }
    readNotifications() {
        const user = this.requireUser();
        this.change(s => { s.notifications.filter(n => n.userId === user.id).forEach(n => { n.read = true; }); });
    }
}

function nextId(items) { return Math.max(0, ...items.map(item => item.id)) + 1; }
function notify(s, userId, actorId, postId, kind, commentId = null) {
    if (userId === actorId) return;
    s.notifications.push({ id: nextId(s.notifications), userId, actorId, postId, kind, commentId, read: false });
    s.notifications = s.notifications.slice(-1000);
}
