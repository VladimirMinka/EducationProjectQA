import { api, formatMoney } from "../api.js";
import { requireSession } from "../auth.js";
import { renderShell, showToast, setCartCount } from "../shell.js";
import { productImageAttrs, bindImageFallbacks } from "../images.js";

export async function renderCart(root) {
  const session = requireSession();
  if (!session) return;

  renderShell(root, { active: "cart", user: session.user });
  const content = root.querySelector("[data-page-content]");
  content.innerHTML = `<p class="loading" data-testid="cart-loading">Загрузка корзины…</p>`;

  let productsById = new Map();
  let cart = null;
  let promoCode = "";
  let notice = "";
  let busy = false;

  try {
    const [list, cartRes] = await Promise.all([
      api.listProducts(),
      api.getCart(session.user.id, session.accessToken),
    ]);
    productsById = new Map((list.products || []).map((p) => [p.id, p]));
    cart = cartRes;
    setCartCount(sumQty(cart.items));
  } catch (err) {
    content.innerHTML = `<div class="alert alert-error" data-testid="cart-error">${escapeHtml(err.message)}</div>`;
    return;
  }

  function paint() {
    const items = cart?.items || [];
    setCartCount(sumQty(items));

    content.innerHTML = `
      <div class="page-head" data-testid="cart-header">
        <div>
          <h1 data-testid="cart-title">Корзина</h1>
          <p data-testid="cart-subtitle">${items.length ? `${sumQty(items)} шт.` : "Пока пусто"}</p>
        </div>
      </div>
      ${notice ? `<div class="alert alert-ok" data-testid="cart-notice" data-order-notice="true">${escapeHtml(notice)}</div>` : ""}
      <div class="cart-layout" data-testid="cart-page" data-page="cart" data-items-count="${items.length}">
        <div class="cart-list" data-testid="cart-list">
          ${
            items.length
              ? items.map((item) => cartItemRow(item, productsById)).join("")
              : `<div class="empty" data-testid="cart-empty">Корзина пуста. <a href="#/catalog" data-testid="cart-empty-catalog-link">Перейти в каталог</a></div>`
          }
        </div>
        <aside class="panel summary" data-testid="cart-summary">
          <div class="summary-row" data-testid="cart-row-subtotal">
            <span>Сабтотал</span>
            <span data-testid="cart-subtotal" data-cents="${Number(cart.subtotalCents ?? 0)}">${formatMoney(cart.subtotalCents)}</span>
          </div>
          <div class="summary-row" data-testid="cart-row-discount">
            <span>Скидка</span>
            <span data-testid="cart-discount" data-cents="${Number(cart.discountCents ?? 0)}">${formatMoney(cart.discountCents)}</span>
          </div>
          <div class="summary-row total" data-testid="cart-row-total">
            <span>Итого</span>
            <span data-testid="cart-total" data-cents="${Number(cart.totalPriceCents ?? 0)}">${formatMoney(cart.totalPriceCents)}</span>
          </div>
          <div data-testid="cart-tags">
            ${
              cart.appliedPromocode
                ? `<span class="tag" data-testid="applied-promo" data-promo="${escapeAttr(cart.appliedPromocode)}">Промо: ${escapeHtml(cart.appliedPromocode)}</span>`
                : ""
            }
            ${
              cart.comboDiscountApplied
                ? `<span class="tag" data-testid="combo-flag" data-combo="true">Combo 10%</span>`
                : ""
            }
          </div>
          ${
            items.length && cart.expiresAt
              ? `<p class="hint" data-testid="cart-expires" data-expires-at="${escapeAttr(cart.expiresAt)}">Корзина истечёт: ${escapeHtml(formatExpires(cart.expiresAt))}</p>`
              : ""
          }
          <div class="promo-row" data-testid="promo-row">
            <input
              type="text"
              placeholder="Промокод"
              value="${escapeAttr(promoCode)}"
              data-testid="promo-input"
              ${busy ? "disabled" : ""}
            />
            <button type="button" class="btn btn-ghost" data-action="apply-promo" data-testid="promo-apply" ${busy || !items.length ? "disabled" : ""}>OK</button>
          </div>
          ${
            cart.appliedPromocode
              ? `<button type="button" class="btn btn-ghost" data-action="clear-promo" data-testid="promo-clear" ${busy ? "disabled" : ""}>Снять промокод</button>`
              : ""
          }
          <button type="button" class="btn btn-primary" data-action="checkout" data-testid="checkout-button" ${busy || !items.length ? "disabled" : ""}>Оформить заказ</button>
          <button type="button" class="btn btn-danger" data-action="clear-cart" data-testid="clear-cart-button" ${busy || !items.length ? "disabled" : ""}>Очистить</button>
        </aside>
      </div>
    `;

    bindImageFallbacks(content);

    content.querySelector("[data-testid=promo-input]")?.addEventListener("input", (e) => {
      promoCode = e.target.value;
    });

    content.querySelectorAll("[data-remove]").forEach((btn) => {
      btn.addEventListener("click", () => run(() => removeItem(btn.dataset.remove)));
    });
    content.querySelector("[data-action=apply-promo]")?.addEventListener("click", () =>
      run(applyPromo)
    );
    content.querySelector("[data-action=clear-promo]")?.addEventListener("click", () =>
      run(clearPromo)
    );
    content.querySelector("[data-action=clear-cart]")?.addEventListener("click", () =>
      run(clearCart)
    );
    content.querySelector("[data-action=checkout]")?.addEventListener("click", () =>
      run(checkout)
    );
  }

  async function run(fn) {
    if (busy) return;
    busy = true;
    paint();
    try {
      await fn();
    } catch (err) {
      showToast(err.message || "Ошибка", true);
    } finally {
      busy = false;
      paint();
    }
  }

  async function removeItem(productId) {
    cart = await api.removeItem(session.user.id, session.accessToken, productId);
    showToast("Товар удалён");
  }

  async function applyPromo() {
    const code = promoCode.trim();
    if (!code) {
      showToast("Введите промокод", true);
      return;
    }
    cart = await api.applyPromocode(session.user.id, session.accessToken, code);
    promoCode = "";
    showToast(`Промокод ${cart.appliedPromocode || code} применён`);
  }

  async function clearPromo() {
    cart = await api.clearPromocode(session.user.id, session.accessToken);
    showToast("Промокод снят");
  }

  async function clearCart() {
    cart = await api.clearCart(session.user.id, session.accessToken);
    notice = "";
    showToast("Корзина очищена");
  }

  async function checkout() {
    const res = await api.createOrder(session.user.id, session.accessToken);
    const orderId = res?.order?.id || "—";
    cart = await api.getCart(session.user.id, session.accessToken);
    notice = `Заказ создан: ${orderId}`;
    showToast("Заказ оформлен");
  }

  paint();
}

function cartItemRow(item, productsById) {
  const product = productsById.get(item.productId) || {
    id: item.productId,
    brand: "",
    name: item.productId,
  };
  const name = product?.name || item.productId;
  const unit = product?.priceCents;
  const qty = Number(item.quantity || 0);
  const line = unit != null ? Number(unit) * qty : null;
  const img = productImageAttrs(product, { alt: name });
  const suffix = String(item.productId || "").replace(/-/g, "").slice(-4).toLowerCase();

  return `
    <article
      class="cart-item"
      data-testid="cart-item"
      data-product-id="${escapeAttr(item.productId)}"
      data-product-suffix="${suffix}"
      data-quantity="${qty}"
    >
      <img
        class="cart-thumb"
        src="${escapeAttr(img.src)}"
        alt="${escapeAttr(img.alt)}"
        data-fallbacks='${escapeAttr(img["data-fallbacks"])}'
        data-testid="cart-item-image"
        loading="lazy"
      />
      <div>
        <h3 data-testid="cart-item-name">${escapeHtml(name)}</h3>
        <div class="cart-meta" data-testid="cart-item-meta">
          <span data-testid="cart-item-brand">${escapeHtml(product?.brand || "")}</span>
          · qty <span data-testid="cart-item-qty">${qty}</span>
          ${line != null ? `· <span data-testid="cart-item-line-total">${formatMoney(line)}</span>` : ""}
        </div>
      </div>
      <button
        type="button"
        class="btn btn-ghost"
        data-remove="${escapeAttr(item.productId)}"
        data-testid="remove-item"
        data-product-id="${escapeAttr(item.productId)}"
      >Удалить</button>
    </article>
  `;
}

function sumQty(items) {
  return (items || []).reduce((n, i) => n + Number(i.quantity || 0), 0);
}

function formatExpires(value) {
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString("ru-RU");
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function escapeAttr(value) {
  return escapeHtml(value).replaceAll("'", "&#39;");
}
