import { getSession } from "./auth.js";
import { initTheme } from "./theme.js";
import { renderLogin } from "./pages/login.js";
import { renderCatalog } from "./pages/catalog.js";
import { renderCart } from "./pages/cart.js";

initTheme();

const app = document.getElementById("app");

function route() {
  const hash = location.hash.replace(/^#/, "") || "/";
  const path = hash.startsWith("/") ? hash : `/${hash}`;

  if (path.startsWith("/login")) {
    renderLogin(app);
    return;
  }
  if (path.startsWith("/cart")) {
    renderCart(app);
    return;
  }
  if (path.startsWith("/catalog") || path === "/") {
    if (!getSession()) {
      location.hash = "#/login";
      return;
    }
    renderCatalog(app);
    return;
  }

  location.hash = getSession() ? "#/catalog" : "#/login";
}

window.addEventListener("hashchange", route);

if (!location.hash) {
  location.hash = getSession() ? "#/catalog" : "#/login";
} else {
  route();
}
