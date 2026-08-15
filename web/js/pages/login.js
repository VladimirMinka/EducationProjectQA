import { api } from "../api.js";
import { getSession, setSession } from "../auth.js";
import { getTheme, toggleTheme } from "../theme.js";

export function renderLogin(root) {
  if (getSession()) {
    location.hash = "#/catalog";
    return;
  }

  let mode = "login";
  let error = "";
  let busy = false;

  function paint() {
    root.innerHTML = `
      <main class="auth-page" data-testid="login-page" data-auth-mode="${mode}">
        <button
          type="button"
          class="theme-fab linkish"
          data-testid="theme-toggle"
          data-theme-current="${getTheme()}"
        >${getTheme() === "dark" ? "Светлая тема" : "Тёмная тема"}</button>
        <div class="auth-stage" data-testid="auth-stage">
          <h1 class="auth-brand" data-testid="auth-brand">STORE</h1>
          <p class="auth-lead" data-testid="auth-lead">Техника Apple, Samsung, NVIDIA и AMD — учебный магазин для API и UI.</p>
          <div class="panel" data-testid="auth-panel">
            <div class="tabs" role="tablist" data-testid="auth-tabs">
              <button type="button" class="tab ${mode === "login" ? "active" : ""}" data-mode="login" data-testid="tab-login" role="tab" aria-selected="${mode === "login"}">Вход</button>
              <button type="button" class="tab ${mode === "register" ? "active" : ""}" data-mode="register" data-testid="tab-register" role="tab" aria-selected="${mode === "register"}">Регистрация</button>
            </div>
            ${error ? `<div class="alert alert-error" data-testid="auth-error" role="alert">${escapeHtml(error)}</div>` : ""}
            <form id="auth-form" data-testid="auth-form" data-auth-mode="${mode}">
              ${
                mode === "register"
                  ? `<div class="field">
                      <label for="name">Имя</label>
                      <input id="name" name="name" autocomplete="name" required data-testid="register-name" />
                    </div>`
                  : ""
              }
              <div class="field">
                <label for="email">Email</label>
                <input id="email" name="email" type="email" autocomplete="username" required data-testid="login-email" />
              </div>
              <div class="field">
                <label for="password">Пароль</label>
                <input id="password" name="password" type="password" autocomplete="${mode === "login" ? "current-password" : "new-password"}" required minlength="6" data-testid="login-password" />
              </div>
              <button class="btn btn-primary" type="submit" ${busy ? "disabled" : ""} data-testid="auth-submit" data-auth-action="${mode}">
                ${busy ? "…" : mode === "login" ? "Войти" : "Создать аккаунт"}
              </button>
            </form>
            <p class="hint" data-testid="auth-hint">Сидовый админ: <code data-testid="seed-admin-email">admin@store.local</code> / <code data-testid="seed-admin-password">admin123</code></p>
          </div>
        </div>
      </main>
    `;

    root.querySelector('[data-testid="theme-toggle"]')?.addEventListener("click", () => {
      toggleTheme();
      paint();
    });

    root.querySelectorAll("[data-mode]").forEach((btn) => {
      btn.addEventListener("click", () => {
        mode = btn.dataset.mode;
        error = "";
        paint();
      });
    });

    root.querySelector("#auth-form")?.addEventListener("submit", onSubmit);
  }

  async function onSubmit(event) {
    event.preventDefault();
    if (busy) return;
    error = "";
    busy = true;
    paint();

    const form = new FormData(event.target);
    const email = String(form.get("email") || "").trim();
    const password = String(form.get("password") || "");
    const name = String(form.get("name") || "").trim();

    try {
      const auth =
        mode === "login"
          ? await api.login({ email, password })
          : await api.register({ email, password, name });
      setSession(auth);
      location.hash = "#/catalog";
    } catch (err) {
      error = err.message || "Не удалось войти";
      busy = false;
      paint();
    }
  }

  paint();
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}
