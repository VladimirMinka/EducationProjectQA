import { clearSession } from "./auth.js";
import { getTheme, toggleTheme } from "./theme.js";

let toastTimer = null;

export function showToast(message, isError = false) {
  document.querySelector(".toast")?.remove();
  const el = document.createElement("div");
  el.className = "toast";
  el.setAttribute("data-testid", "toast");
  el.setAttribute("data-toast-type", isError ? "error" : "ok");
  if (isError) el.style.borderColor = "rgba(255, 107, 122, 0.45)";
  el.textContent = message;
  document.body.appendChild(el);
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => el.remove(), 2800);
}

export function setCartCount(count) {
  const badge = document.querySelector('[data-testid="cart-badge"]');
  if (!badge) return;
  const n = Number(count) || 0;
  badge.hidden = n <= 0;
  badge.textContent = String(n);
  badge.setAttribute("data-count", String(n));
}

export function renderShell(root, { active, user }) {
  const theme = getTheme();
  root.innerHTML = `
    <div class="shell" data-testid="app-shell" data-active-page="${active}">
      <header class="topbar" data-testid="topbar">
        <a class="brand" href="#/catalog" data-testid="brand-link">
          <span class="brand-mark" aria-hidden="true"></span>
          <span data-testid="brand-name">STORE</span>
        </a>
        <nav class="nav" data-testid="main-nav">
          <a
            href="#/catalog"
            class="${active === "catalog" ? "active" : ""}"
            data-testid="nav-catalog"
            data-nav="catalog"
            aria-current="${active === "catalog" ? "page" : "false"}"
          >Каталог</a>
          <a
            href="#/cart"
            class="cart-link ${active === "cart" ? "active" : ""}"
            data-testid="nav-cart"
            data-nav="cart"
            aria-current="${active === "cart" ? "page" : "false"}"
          >
            Корзина
            <span class="badge" data-testid="cart-badge" data-count="0" hidden>0</span>
          </a>
          <button
            type="button"
            class="linkish theme-toggle"
            data-testid="theme-toggle"
            data-theme-current="${theme}"
            title="Тема"
          >
            ${theme === "dark" ? "Светлая" : "Тёмная"}
          </button>
          <span class="user-label" data-testid="user-name">${escapeHtml(user?.name || user?.email || "")}</span>
          <button type="button" class="linkish" data-testid="logout-button">Выйти</button>
        </nav>
      </header>
      <div data-page-content data-testid="page-content"></div>
    </div>
  `;

  root.querySelector('[data-testid="theme-toggle"]')?.addEventListener("click", () => {
    const next = toggleTheme();
    const btn = root.querySelector('[data-testid="theme-toggle"]');
    if (btn) {
      btn.textContent = next === "dark" ? "Светлая" : "Тёмная";
      btn.setAttribute("data-theme-current", next);
    }
  });

  root.querySelector('[data-testid="logout-button"]')?.addEventListener("click", () => {
    clearSession();
    location.hash = "#/login";
  });
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}
