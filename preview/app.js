import { PreviewStore, safeImage } from './store.js';

const main = document.querySelector('#content');
const sidebar = document.querySelector('#sidebar');
const viewer = document.querySelector('#viewer');
const notice = document.querySelector('#notice');
const escape = value => String(value).replace(/[&<>"']/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[char]);
const icon = name => `<svg class="icon" aria-hidden="true"><use href="./static/icons.svg#${name}"></use></svg>`;
const empty = message => `<p class="preview-empty">${escape(message)}</p>`;
const heading = (title, subtitle = '') => `<div class="page-heading"><span class="eyebrow">THE COMMON ROOM</span><h1>${escape(title)}</h1>${subtitle ? `<p>${escape(subtitle)}</p>` : ''}</div>`;
const date = value => escape(new Date(value).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' }));
const button = (action, kind, id, label, extra = '') => `<button type="button" class="preview-button" data-action="${action}" data-kind="${kind}" data-id="${id}" ${extra}>${label}</button>`;
let store;
let noticeTimer;

function announce(message) {
    clearTimeout(noticeTimer);
    notice.textContent = message;
    notice.hidden = false;
    noticeTimer = setTimeout(() => { notice.hidden = true; }, 7000);
}

function route() {
    const raw = location.hash.slice(1) || '/';
    const split = raw.indexOf('?');
    return { path: split < 0 ? raw : raw.slice(0, split), query: new URLSearchParams(split < 0 ? '' : raw.slice(split + 1)) };
}

function go(path) {
    if (location.hash === `#${path}`) render(true);
    else location.hash = path;
}

function metadata(post) {
    return `<div class="post-meta"><span>${icon('user')} ${escape(store.name(post.userId))}</span><span class="categories">${escape(post.topics.join(', '))}</span><span>${date(post.createdAt)}</span></div>`;
}

function postCard(post) {
    const { likes, dislikes } = store.counts('post', post.id);
    return `<article class="post-card"><div class="post-header"><h3><a href="#/posts/${post.id}">${escape(post.title)}</a></h3>${metadata(post)}</div>
        ${post.status === 'pending' ? '<p class="form-hint">Awaiting moderator approval</p>' : ''}
        <div class="post-content">${post.image ? `<div class="post-image"><img src="${escape(post.image)}" alt="Attachment to ${escape(post.title)}" loading="lazy"></div>` : ''}<p class="post-text">${escape(post.content)}</p></div>
        <div class="post-footer"><div class="post-stats"><span>${icon('thumbs-up')} ${likes}</span><span>${icon('thumbs-down')} ${dislikes}</span><a href="#/posts/${post.id}">Join the conversation →</a></div></div></article>`;
}

function reactionButtons(kind, item) {
    const counts = store.counts(kind, item.id);
    const mine = store.state.reactions.find(r => r.kind === kind && r.targetId === item.id && r.userId === store.user?.id);
    const post = kind === 'post' ? item : store.state.posts.find(p => p.id === item.postId);
    const disabled = !store.user || item.status !== 'approved' || post.status !== 'approved' ? 'disabled' : '';
    return `<div class="preview-actions">
        ${button('like', kind, item.id, `${icon('thumbs-up')} ${counts.likes} likes`, `aria-label="Like ${kind}" aria-pressed="${mine?.like === true}" ${disabled}`)}
        ${button('dislike', kind, item.id, `${icon('thumbs-down')} ${counts.dislikes} dislikes`, `aria-label="Dislike ${kind}" aria-pressed="${mine?.like === false}" ${disabled}`)}
        </div>`;
}

function contentActions(kind, item) {
    const owner = item.userId === store.user?.id;
    const canDelete = owner || (kind === 'post' ? store.isStaff() : store.user?.role === 'admin');
    return `<div class="preview-actions">${owner ? `<a class="preview-button" href="#/${kind === 'post' ? 'posts' : 'comments'}/${item.id}/edit">Edit ${kind === 'post' ? 'discussion' : 'comment'}</a>` : ''}
        ${canDelete ? button('delete', kind, item.id, `Delete ${kind === 'post' ? 'discussion' : 'comment'}`) : ''}</div>`;
}

function feed(path, query) {
    const selected = query.getAll('topics').filter(t => store.state.topics.includes(t));
    let title = 'Latest discussions';
    let posts = store.state.posts.filter(p => p.status === 'approved');
    if (path === '/my-posts') {
        title = 'My discussions';
        posts = store.state.posts.filter(p => p.userId === store.user.id);
    } else if (path === '/liked-posts') {
        title = 'Liked discussions';
        const liked = store.state.reactions.filter(r => r.userId === store.user.id && r.kind === 'post' && r.like).map(r => r.targetId);
        posts = posts.filter(p => liked.includes(p.id));
    } else if (selected.length) {
        title = `Discussions in ${selected.join(', ')}`;
        posts = posts.filter(p => p.topics.some(t => selected.includes(t)));
    }
    posts.sort((a, b) => b.createdAt - a.createdAt || b.id - a.id);
    return `${heading('A little space for big ideas.', 'Share perspectives. Ask questions. Find your next conversation.')}
        <div class="feed-heading"><h2>${escape(title)}</h2><span>${posts.length} discussions</span></div>${posts.map(postCard).join('') || empty('No discussions here yet.')}`;
}

function discussion(id) {
    const post = store.state.posts.find(p => p.id === id);
    if (!store.canView(post)) return notFound();
    const comments = store.state.comments.filter(c => c.postId === id && store.canViewComment(c)).sort((a, b) => a.createdAt - b.createdAt || a.id - b.id);
    return `<article class="post"><h1>${escape(post.title)}</h1>${metadata(post)}
        ${post.status === 'pending' ? '<p class="preview-help">Awaiting approval. Visible only to its author and preview staff.</p>' : ''}
        ${post.image ? `<div class="post-image"><img src="${escape(post.image)}" alt="Attachment to ${escape(post.title)}"></div>` : ''}
        <div class="post-content">${escape(post.content)}</div>${reactionButtons('post', post)}${contentActions('post', post)}</article>
        <h2 class="comments-name">The conversation</h2>
        ${comments.map(c => `<article class="comment"><p class="comment-meta">${escape(store.name(c.userId))} · ${date(c.createdAt)}</p><p class="comment-content">${escape(c.content)}</p>${c.status === 'pending' ? '<p class="form-hint">Awaiting moderator approval</p>' : ''}${reactionButtons('comment', c)}${contentActions('comment', c)}</article>`).join('') || empty('No comments yet. Add your perspective.')}
        ${post.status === 'approved' ? `<section class="add-comment"><h2>Add your perspective</h2>${store.user ? `<form class="preview-form" data-form="comment" data-id="${id}"><p class="error-message" role="alert" hidden></p><label>Your comment<textarea name="content" rows="4" maxlength="20000" required></textarea></label><p class="form-hint">${store.state.premoderate && store.user.role === 'user' ? 'Your comment will await simulated staff approval.' : 'Saved only in this browser.'}</p><button class="preview-button primary" type="submit">Submit comment</button></form>` : '<p>Choose a fictional account above to comment or react.</p>'}</section>` : ''}`;
}

function postForm(id = null) {
    const post = id ? store.state.posts.find(p => p.id === id) : null;
    if (id && (!post || post.userId !== store.user.id)) return notFound();
    return `${heading(id ? 'Edit your discussion.' : 'Start a conversation.', 'Your changes are private to this browser.')}
        <section class="post-container"><form class="preview-form" data-form="post" data-id="${id ?? ''}">
        <p class="error-message" role="alert" hidden></p>
        <label>Title<input name="title" maxlength="200" value="${escape(post?.title ?? '')}" required></label>
        <label>Your perspective<textarea name="content" rows="7" maxlength="20000" required>${escape(post?.content ?? '')}</textarea></label>
        <fieldset><legend>Topics (choose at least one)</legend><div class="category-options">${store.state.topics.map(t => `<label><input type="checkbox" name="topics" value="${escape(t)}" ${post?.topics.includes(t) ? 'checked' : ''}>${escape(t)}</label>`).join('')}</div></fieldset>
        <label>Optional image<input type="file" name="image" accept="image/png,image/jpeg,image/gif,image/webp"></label>
        <p class="form-hint">PNG, JPEG, GIF, or WebP up to 1 MiB. Preview attachments stay in this browser.</p>
        ${post?.image ? `<img class="preview-thumbnail" src="${escape(post.image)}" alt="Current attachment"><label><input name="remove-image" type="checkbox"> Remove current attachment</label>` : ''}
        <p class="form-hint">${store.state.premoderate && store.user.role === 'user' ? 'Member submissions await simulated staff approval.' : 'Published immediately in your local preview.'}</p>
        <div class="preview-actions"><button class="preview-button primary" type="submit">${id ? 'Save discussion' : 'Publish discussion'}</button><a href="${id ? `#/posts/${id}` : '#/'}">Cancel</a></div>
        </form></section>`;
}

function commentForm(id) {
    const comment = store.state.comments.find(c => c.id === id);
    if (!comment || comment.userId !== store.user.id) return notFound();
    return `${heading('Edit your comment.')}<section class="post-container"><form class="preview-form" data-form="edit-comment" data-id="${id}" data-post="${comment.postId}"><p class="error-message" role="alert" hidden></p><label>Your comment<textarea name="content" rows="5" maxlength="20000" required>${escape(comment.content)}</textarea></label><div class="preview-actions"><button class="preview-button primary" type="submit">Save comment</button><a href="#/posts/${comment.postId}">Cancel</a></div></form></section>`;
}

function profile() {
    const user = store.user;
    return `${heading('Make yourself at home.', 'Explore with a fictional persona. No password or real account is required.')}
        <section class="profile-container"><h2>${escape(user.name)}</h2><div class="profile-info"><p><strong>Preview role:</strong> ${escape(user.role)}</p><p><strong>Discussions:</strong> ${store.state.posts.filter(p => p.userId === user.id).length}</p><p><strong>Comments:</strong> ${store.state.comments.filter(c => c.userId === user.id).length}</p></div>
        <div class="preview-settings"><label for="role">Try a simulated role<select id="role">${[['user', 'Member'], ['moderator', 'Moderator'], ['admin', 'Administrator']].map(([role, label]) => `<option value="${role}" ${user.role === role ? 'selected' : ''}>${label}</option>`).join('')}</select></label></div>
        <p class="preview-help">Roles here only change this browser simulation. The original application checks permissions on its Go server. This preview includes the content approval queue; moderator requests, reports, and topic administration are available in the full application.</p>
        <div class="preview-actions"><a href="#/activity" class="preview-button">My activity</a><a href="#/notifications" class="preview-button">Notifications (${store.unread()})</a><a href="#/moderation" class="preview-button">Community tools</a></div></section>`;
}

function activity() {
    const posts = store.state.posts.filter(p => p.userId === store.user.id).sort((a, b) => b.createdAt - a.createdAt);
    const comments = store.state.comments.filter(c => c.userId === store.user.id);
    const reactions = store.state.reactions.filter(r => r.userId === store.user.id);
    return `${heading('Your conversations.', 'Discussions, comments, and current reactions in this browser.')}
        <section class="moderation-section"><h2>Your discussions</h2>${posts.map(postCard).join('') || empty('You have not started a discussion.')}</section>
        <section class="moderation-section"><h2>Your comments</h2>${comments.map(c => {
            const post = store.state.posts.find(p => p.id === c.postId);
            return `<article class="moderation-item">${store.canView(post) ? `<a href="#/posts/${post.id}"><strong>${escape(post.title)}</strong></a>` : '<strong>Discussion awaiting review</strong>'}<p class="preview-body">${escape(c.content)}</p><p class="form-hint">${escape(c.status)}</p>${contentActions('comment', c)}</article>`;
        }).join('') || empty('You have not commented yet.')}</section>
        <section class="moderation-section"><h2>Your reactions</h2>${reactions.map(r => {
            const comment = r.kind === 'comment' ? store.state.comments.find(c => c.id === r.targetId) : null;
            const post = store.state.posts.find(p => p.id === (comment?.postId ?? r.targetId));
            if (post.status !== 'approved' || (comment && comment.status !== 'approved')) return '';
            return `<p class="moderation-item">${r.like ? 'Liked' : 'Disliked'} ${comment ? 'a comment on' : ''} <a href="#/posts/${post.id}">${escape(post.title)}</a></p>`;
        }).filter(Boolean).join('') || empty('No reactions on published content yet.')}</section>`;
}

function notifications() {
    const items = store.state.notifications.filter(n => n.userId === store.user.id && store.canView(store.state.posts.find(p => p.id === n.postId))).sort((a, b) => b.id - a.id).slice(0, 100);
    return `${heading('The conversation continues.', `${store.unread()} unread · showing your latest 100 notifications.`)}
        <section class="moderation-section"><h2>On your discussions</h2>${store.unread() ? button('read', '', 0, 'Mark all as read') : ''}
        ${items.map(n => `<article class="moderation-item ${n.read ? '' : 'notification-unread'}"><p><strong>${escape(store.name(n.actorId))}</strong> ${n.kind}${n.kind === 'commented' ? ' on' : ''} your discussion <a href="#/posts/${n.postId}">${escape(store.state.posts.find(p => p.id === n.postId).title)}</a>.</p>${n.read ? '' : '<p class="form-hint">Unread</p>'}</article>`).join('') || empty('No notifications yet. Switch to another account and react or comment on one of your discussions.')}</section>`;
}

function moderation() {
    if (!store.isStaff()) return `${heading('Community tools.', 'Try the content approval workflow with a simulated staff role.')}<section class="moderation-section"><p>Choose Moderator or Administrator on your preview profile to review content.</p><div class="preview-actions"><a class="preview-button" href="#/profile">Choose a preview role</a></div></section>`;
    const items = [...store.state.posts.filter(p => p.status === 'pending').map(p => ({ ...p, kind: 'post' })), ...store.state.comments.filter(c => c.status === 'pending').map(c => ({ ...c, kind: 'comment' }))];
    return `${heading('Community tools.', 'A simulated approval queue. All changes apply only to your browser.')}
        <section class="moderation-section"><h2>Publication settings</h2><div class="preview-settings"><label><input id="premoderate" type="checkbox" ${store.state.premoderate ? 'checked' : ''}> Require approval for member posts and comments</label></div><p class="preview-help">Enable review, switch to a member, and submit content. Switch back to staff to approve it. Existing published content stays published.</p></section>
        <section class="moderation-section"><h2>Awaiting approval (${items.length})</h2>${items.map(item => `<article class="moderation-item"><h3>${escape(item.title ?? 'Comment')}</h3><p class="preview-date">${escape(store.name(item.userId))} · ${item.kind}</p><p class="preview-body">${escape(item.content)}</p><div class="preview-actions"><a href="#/posts/${item.kind === 'post' ? item.id : item.postId}">View discussion</a>${button('approve', item.kind, item.id, 'Approve')}${button('reject', item.kind, item.id, 'Reject')}</div></article>`).join('') || empty('Nothing is awaiting approval.')}</section>`;
}

function notFound() { return `${heading('Discussion unavailable.', 'It may have been removed or may be awaiting approval.')}<a class="preview-button" href="#/">Return to the community</a>`; }
function guestPage() { return `${heading('Choose a demo account.', 'Use “Explore as” above to try this part of the forum.')}<p class="preview-help">These fictional personas simulate sign-in. No real authentication occurs.</p>`; }

function render(focus = false) {
    const { path, query } = route();
    viewer.innerHTML = `<option value="">Guest</option>${store.state.users.map(u => `<option value="${u.id}">${escape(u.name)}</option>`).join('')}`;
    viewer.value = store.state.viewer ?? '';
    document.querySelector('#member-links').innerHTML = store.user ? '<a href="#/profile">Profile</a><a href="#/posts/new" class="create-post-icon">New discussion</a>' : '<a href="#/profile">Explore the demo</a>';
    const nav = [['/', 'Home'], ['/activity', 'My activity'], ['/notifications', `Notifications (${store.unread()})`], ['/my-posts', 'My discussions'], ['/liked-posts', 'Liked discussions'], ['/moderation', 'Community tools']];
    sidebar.innerHTML = `<div class="workspace-label">${icon('layer-group')}<div>The common room<small>Your everyday community</small></div></div>
        <nav class="preview-nav"><ul><li><h2 class="eyebrow sidebar-section-label">EXPLORE TOPICS</h2><form class="category-filter" data-form="filter"><div class="checkbox-column">${store.state.topics.map(t => `<div class="checkbox-item"><input type="checkbox" id="topic-${escape(t)}" name="topics" value="${escape(t)}" ${query.getAll('topics').includes(t) ? 'checked' : ''}><label for="topic-${escape(t)}">${escape(t)}</label></div>`).join('')}</div><div class="button-group"><button class="filter-button" type="submit">Apply</button><a class="preview-button" href="#/">Clear</a></div></form></li>${nav.filter(([url]) => store.user || url === '/').map(([url, label]) => `<li><a href="#${url}" ${path === url ? 'aria-current="page"' : ''}>${escape(label)}</a></li>`).join('')}</ul></nav><div class="sidebar-note"><span class="tiny-star" aria-hidden="true">✳</span><p>Good conversations<br>start with curiosity.</p><span>Bring yours.</span></div>`;
    const postMatch = path.match(/^\/posts\/(\d+)$/);
    const editPost = path.match(/^\/posts\/(\d+)\/edit$/);
    const editComment = path.match(/^\/comments\/(\d+)\/edit$/);
    const privateRoute = ['/profile', '/activity', '/notifications', '/moderation', '/my-posts', '/liked-posts', '/posts/new'].includes(path) || editPost || editComment;
    if (privateRoute && !store.user) main.innerHTML = guestPage();
    else if (['/', '/my-posts', '/liked-posts'].includes(path)) main.innerHTML = feed(path, query);
    else if (postMatch) main.innerHTML = discussion(Number(postMatch[1]));
    else if (editPost) main.innerHTML = postForm(Number(editPost[1]));
    else if (editComment) main.innerHTML = commentForm(Number(editComment[1]));
    else if (path === '/posts/new') main.innerHTML = postForm();
    else if (path === '/profile') main.innerHTML = profile();
    else if (path === '/activity') main.innerHTML = activity();
    else if (path === '/notifications') main.innerHTML = notifications();
    else if (path === '/moderation') main.innerHTML = moderation();
    else main.innerHTML = notFound();
    document.title = `${main.querySelector('h1')?.textContent ?? 'Community'} | Yaplane preview`;
    if (focus) { main.focus({ preventScroll: true }); window.scrollTo({ top: 0 }); }
}

async function readImage(file) {
    if (!file?.size) return '';
    if (file.size > 1024 * 1024 || !['image/png', 'image/jpeg', 'image/gif', 'image/webp'].includes(file.type)) throw new Error('Choose a PNG, JPEG, GIF, or WebP image up to 1 MiB.');
    const data = await new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(reader.result);
        reader.onerror = () => reject(new Error('The image could not be read. Please choose another file.'));
        reader.readAsDataURL(file);
    });
    if (!safeImage(data)) throw new Error('The selected file is not a supported image.');
    const image = new Image();
    image.src = data;
    try { await image.decode(); } catch { throw new Error('The selected file could not be decoded as an image.'); }
    return data;
}

function bindEvents() {
    window.addEventListener('hashchange', () => render(true));
    window.addEventListener('storage', event => {
        if (event.key !== 'yaplanePreview.v1' && event.key !== null) return;
        try { store.reload(); render(); announce('Preview updated from another tab.'); } catch (error) { announce(error.message); }
    });
    viewer.addEventListener('change', () => {
        try { store.selectViewer(viewer.value ? Number(viewer.value) : null); render(); announce(store.user ? `Exploring as ${store.user.name}.` : 'Exploring as a guest.'); }
        catch (error) { viewer.value = store.state.viewer ?? ''; announce(error.message); }
    });
    document.querySelector('#reset').addEventListener('click', () => {
        if (!confirm('Reset this browser’s demo? Your local discussions, attachments, and changes will be removed.')) return;
        try { store.reset(); go('/'); announce('The fictional community has been restored.'); } catch (error) { announce(error.message); }
    });
    main.addEventListener('change', event => {
        try {
            if (event.target.id === 'role') { store.setRole(event.target.value); render(); announce('Preview role updated.'); }
            if (event.target.id === 'premoderate') { store.setPremoderate(event.target.checked); render(); announce('Publication settings updated.'); }
        } catch (error) { render(); announce(error.message); }
    });
    main.addEventListener('click', event => {
        const control = event.target.closest('button[data-action]');
        if (!control) return;
        const { action, kind } = control.dataset;
        const id = Number(control.dataset.id);
        try {
            if (action === 'like' || action === 'dislike') store.react(kind, id, action === 'like');
            else if (action === 'read') store.readNotifications();
            else if (action === 'approve') store.approve(kind, id);
            else if (action === 'delete' || action === 'reject') {
                if (!confirm(`${action === 'reject' ? 'Reject' : 'Delete'} this ${kind === 'post' ? 'discussion and its comments' : 'comment'} in your local preview?`)) return;
                store.deleteContent(kind, id, action === 'reject');
                if (kind === 'post' && route().path === `/posts/${id}`) go('/');
            }
            render();
            const replacement = main.querySelector(`button[data-action="${action}"][data-kind="${kind}"][data-id="${id}"]`);
            (replacement ?? main).focus({ preventScroll: true });
            announce(action === 'read' ? 'Notifications marked as read.' : 'Preview updated in this browser.');
        } catch (error) { announce(error.message); }
    });
    document.addEventListener('submit', async event => {
        const form = event.target.closest('form[data-form]');
        if (!form) return;
        event.preventDefault();
        const data = new FormData(form);
        const type = form.dataset.form;
        if (type === 'filter') {
            const query = new URLSearchParams();
            data.getAll('topics').forEach(t => query.append('topics', t));
            go(query.size ? `/?${query}` : '/');
            return;
        }
        const submit = form.querySelector('button[type=submit]');
        if (submit.disabled) return;
        submit.disabled = true;
        const actor = store.state.viewer;
        const originalForm = form;
        try {
            const id = Number(form.dataset.id) || null;
            if (type === 'post') {
                let image = id ? store.state.posts.find(p => p.id === id)?.image ?? '' : '';
                if (data.has('remove-image')) image = '';
                const file = data.get('image');
                if (file?.size) image = await readImage(file);
                if (!form.isConnected || store.state.viewer !== actor) throw new Error('The preview changed while reading the image. Please submit again.');
                const postId = store.savePost({ id, title: data.get('title'), content: data.get('content'), topics: data.getAll('topics'), image });
                go(`/posts/${postId}`);
            } else if (type === 'comment') {
                store.saveComment(id, data.get('content'));
                render();
                main.focus({ preventScroll: true });
            } else if (type === 'edit-comment') {
                store.saveComment(Number(form.dataset.post), data.get('content'), id);
                go(`/posts/${form.dataset.post}`);
            }
            announce('Saved in this browser.');
        } catch (error) {
            const field = originalForm.querySelector('[role=alert]');
            if (field && originalForm.isConnected) { field.textContent = error.message; field.hidden = false; }
            else announce(error.message);
        } finally { submit.disabled = false; }
    });
}

async function start() {
    try {
        const response = await fetch(new URL('./seed.json', import.meta.url));
        if (!response.ok) throw new Error('The fictional demo data is unavailable. Please reload the page.');
        store = new PreviewStore(await response.json());
        if (store.warning) {
            const warning = document.querySelector('#storage-warning');
            warning.textContent = store.warning;
            warning.hidden = false;
        }
        viewer.disabled = false;
        document.querySelector('#reset').disabled = false;
        bindEvents();
        render();
    } catch (error) {
        main.innerHTML = heading('The preview could not start.');
        const message = document.createElement('p');
        message.className = 'error-message';
        message.textContent = error.message;
        main.append(message);
    }
}
start();
