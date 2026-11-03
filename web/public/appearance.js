/* Local, CSP-compatible first-paint preferences. No network or credentials. */
(() => {
  const root = document.documentElement;
  const read = (key, values, fallback) => {
    try { const value = JSON.parse(localStorage.getItem(`flow.ui.${key}`) || 'null'); return values.includes(value) ? value : fallback; }
    catch { return fallback; }
  };
  const theme = read('theme', ['system', 'dark', 'light'], 'system');
  root.dataset.theme = theme === 'system' ? (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light') : theme;
  root.dataset.palette = read('palette', ['iris', 'lagoon', 'rose'], 'iris');
  root.dataset.density = read('density', ['comfortable', 'compact'], 'comfortable');
  root.dataset.motion = read('motion', ['auto', 'reduce'], 'auto');
  root.style.fontSize = `${read('textScale', [100, 125, 150, 200], 100)}%`;
})();
