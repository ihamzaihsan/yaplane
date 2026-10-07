// Apply the saved theme before the stylesheet paints.
(() => {
    const storageKey = 'yaplaneTheme';
    const system = window.matchMedia('(prefers-color-scheme: dark)');
    const root = document.documentElement;
    let preference;
    let button;

    function readPreference() {
        try { return localStorage.getItem(storageKey); } catch { return null; }
    }

    function apply() {
        const dark = preference === 'dark' || (preference !== 'light' && system.matches);
        root.dataset.theme = dark ? 'dark' : 'light';
        if (button) {
            button.setAttribute('aria-pressed', String(dark));
            button.title = dark ? 'Switch to light mode' : 'Switch to dark mode';
            button.querySelector('use').setAttribute('href', `/static/icons.svg#${dark ? 'sun' : 'moon'}`);
        }
    }

    preference = readPreference();
    apply();

    document.addEventListener('DOMContentLoaded', () => {
        const navigation = document.querySelector('header nav ul');
        if (!navigation) return;
        const item = document.createElement('li');
        button = document.createElement('button');
        button.type = 'button';
        button.className = 'theme-toggle';
        button.setAttribute('aria-label', 'Dark mode');
        button.innerHTML = '<svg class="icon" aria-hidden="true"><use></use></svg>';
        item.append(button);
        navigation.prepend(item);
        apply();
        button.addEventListener('click', () => {
            preference = root.dataset.theme === 'dark' ? 'light' : 'dark';
            try { localStorage.setItem(storageKey, preference); } catch { /* Keep the choice for this page. */ }
            apply();
        });
    });

    system.addEventListener('change', apply);
    window.addEventListener('storage', event => {
        if (event.key === storageKey || event.key === null) {
            preference = readPreference();
            apply();
        }
    });
})();
