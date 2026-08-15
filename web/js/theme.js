const THEME_KEY = "store.theme";

export function getTheme() {
  const stored = localStorage.getItem(THEME_KEY);
  if (stored === "light" || stored === "dark") return stored;
  return "dark";
}

export function applyTheme(theme) {
  const next = theme === "light" ? "light" : "dark";
  document.documentElement.dataset.theme = next;
  localStorage.setItem(THEME_KEY, next);
  return next;
}

export function initTheme() {
  return applyTheme(getTheme());
}

export function toggleTheme() {
  return applyTheme(getTheme() === "dark" ? "light" : "dark");
}
