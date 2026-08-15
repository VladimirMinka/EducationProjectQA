import { api, formatMoney } from "../api.js";
import { requireSession } from "../auth.js";
import { renderShell, showToast, setCartCount } from "../shell.js";
import { productImageAttrs, bindImageFallbacks } from "../images.js";

const BRANDS = [
  { id: "all", label: "Все" },
  { id: "apple", label: "Apple" },
  { id: "samsung", label: "Samsung" },
  { id: "nvidia", label: "NVIDIA" },
  { id: "amd", label: "AMD" },
];

export async function renderCatalog(root) {
  const session = requireSession();
  if (!session) return;

  renderShell(root, { active: "catalog", user: session.user });
  const content = root.querySelector("[data-page-content]");
  content.innerHTML = `<p class="loading" data-testid="catalog-loading">Загрузка каталога…</p>`;

  let brand = "all";
  let products = [];
  let priceFloor = 0;
  let priceCeil = 0;
  let priceMin = 0;
  let priceMax = 0;

  try {
    const [list, cart] = await Promise.all([
      api.listProducts(),
      api.getCart(session.user.id, session.accessToken).catch(() => null),
    ]);
    products = list.products || [];
    const cents = products.map((p) => Number(p.priceCents) || 0);
    priceFloor = cents.length ? Math.min(...cents) : 0;
    priceCeil = cents.length ? Math.max(...cents) : 0;
    priceMin = priceFloor;
    priceMax = priceCeil;
    setCartCount(sumQty(cart?.items));
  } catch (err) {
    content.innerHTML = `<div class="alert alert-error" data-testid="catalog-error">${escapeHtml(err.message)}</div>`;
    return;
  }

  function filteredProducts() {
    return products.filter((p) => {
      const brandOk = brand === "all" || p.brand === brand;
      const price = Number(p.priceCents) || 0;
      return brandOk && price >= priceMin && price <= priceMax;
    });
  }

  function brandCounts() {
    const map = {};
    for (const b of BRANDS) {
      if (b.id === "all") {
        map.all = products.filter((p) => {
          const price = Number(p.priceCents) || 0;
          return price >= priceMin && price <= priceMax;
        }).length;
        continue;
      }
      map[b.id] = products.filter((p) => {
        const price = Number(p.priceCents) || 0;
        return p.brand === b.id && price >= priceMin && price <= priceMax;
      }).length;
    }
    return map;
  }

  function paintShell() {
    const step = Math.max(100, Math.round((priceCeil - priceFloor) / 100) || 100);

    content.innerHTML = `
      <div class="catalog-layout" data-testid="catalog-layout" data-page="catalog">
        <aside class="filters-panel" data-testid="catalog-filters">
          <h2 class="filters-title" data-testid="filters-title">Фильтры</h2>

          <p class="filters-hint" data-testid="filter-brand-label">Бренд</p>
          <div class="filter-list" data-brand-list data-testid="brand-filter-list" role="list"></div>

          <p class="filters-hint filters-hint-spaced" data-testid="filter-price-label">Цена</p>
          <div class="price-filter" data-testid="price-filter">
            <div class="price-labels">
              <span data-testid="price-min-label">${formatMoney(priceMin)}</span>
              <span data-testid="price-max-label">${formatMoney(priceMax)}</span>
            </div>
            <div class="range-wrap" data-testid="price-range">
              <div class="range-track" data-range-track data-testid="price-range-track"></div>
              <input
                type="range"
                class="range-input range-min"
                data-testid="price-min"
                min="${priceFloor}"
                max="${priceCeil}"
                step="${step}"
                value="${priceMin}"
                aria-label="Минимальная цена"
              />
              <input
                type="range"
                class="range-input range-max"
                data-testid="price-max"
                min="${priceFloor}"
                max="${priceCeil}"
                step="${step}"
                value="${priceMax}"
                aria-label="Максимальная цена"
              />
            </div>
            <p class="price-bounds" data-testid="price-bounds">${formatMoney(priceFloor)} — ${formatMoney(priceCeil)}</p>
            <button type="button" class="btn btn-ghost filter-reset" data-testid="price-reset">Сбросить цену</button>
          </div>
        </aside>

        <section class="catalog-main" data-testid="catalog-main">
          <div class="page-head">
            <div>
              <h1 data-testid="catalog-title">Каталог</h1>
              <p data-catalog-summary data-testid="catalog-summary"></p>
            </div>
          </div>
          <div class="grid" data-testid="product-grid"></div>
        </section>
      </div>
    `;

    const minInput = content.querySelector("[data-testid=price-min]");
    const maxInput = content.querySelector("[data-testid=price-max]");

    const syncPrice = (which) => {
      let min = Number(minInput.value);
      let max = Number(maxInput.value);
      if (min > max) {
        if (which === "min") max = min;
        else min = max;
        minInput.value = String(min);
        maxInput.value = String(max);
      }
      priceMin = min;
      priceMax = max;
      updatePriceUI();
      paintResults();
    };

    minInput.addEventListener("input", () => syncPrice("min"));
    maxInput.addEventListener("input", () => syncPrice("max"));

    content.querySelector("[data-testid=price-reset]")?.addEventListener("click", () => {
      priceMin = priceFloor;
      priceMax = priceCeil;
      minInput.value = String(priceMin);
      maxInput.value = String(priceMax);
      updatePriceUI();
      paintResults();
    });

    paintResults();
  }

  function updatePriceUI() {
    const minLabel = content.querySelector("[data-testid=price-min-label]");
    const maxLabel = content.querySelector("[data-testid=price-max-label]");
    const track = content.querySelector("[data-range-track]");
    if (minLabel) minLabel.textContent = formatMoney(priceMin);
    if (maxLabel) maxLabel.textContent = formatMoney(priceMax);
    if (track && priceCeil > priceFloor) {
      const left = ((priceMin - priceFloor) / (priceCeil - priceFloor)) * 100;
      const right = ((priceMax - priceFloor) / (priceCeil - priceFloor)) * 100;
      track.style.left = `${left}%`;
      track.style.width = `${Math.max(0, right - left)}%`;
    }
  }

  function paintResults() {
    const filtered = filteredProducts();
    const c = brandCounts();

    const brandList = content.querySelector("[data-brand-list]");
    if (brandList) {
      brandList.innerHTML = BRANDS.map(
        (b) => `
        <button
          type="button"
          class="filter-item ${brand === b.id ? "active" : ""}"
          data-brand="${b.id}"
          data-testid="filter-${b.id}"
          data-filter-type="brand"
          data-active="${brand === b.id ? "true" : "false"}"
          role="listitem"
        >
          <span data-testid="filter-${b.id}-label">${b.label}</span>
          <span class="filter-count" data-testid="filter-${b.id}-count">${c[b.id] ?? 0}</span>
        </button>`
      ).join("");

      brandList.querySelectorAll("[data-brand]").forEach((btn) => {
        btn.addEventListener("click", () => {
          brand = btn.dataset.brand;
          paintResults();
        });
      });
    }

    const summary = content.querySelector("[data-catalog-summary]");
    if (summary) {
      const bits = [`${filtered.length} товаров`];
      if (brand !== "all") bits.push(brand);
      if (priceMin > priceFloor || priceMax < priceCeil) {
        bits.push(`${formatMoney(priceMin)}–${formatMoney(priceMax)}`);
      }
      summary.textContent = bits.join(" · ");
    }

    const grid = content.querySelector("[data-testid=product-grid]");
    if (grid) {
      grid.innerHTML = filtered.length
        ? filtered.map(productCard).join("")
        : `<div class="empty" data-testid="catalog-empty">Нет товаров для этих фильтров</div>`;
      grid.setAttribute("data-count", String(filtered.length));
      bindImageFallbacks(grid);
      grid.querySelectorAll("[data-add]").forEach((btn) => {
        btn.addEventListener("click", () => addToCart(btn));
      });
    }

    updatePriceUI();
  }

  async function addToCart(btn) {
    const productId = btn.dataset.add;
    btn.disabled = true;
    try {
      const cart = await api.addItem(
        session.user.id,
        session.accessToken,
        productId,
        1
      );
      setCartCount(sumQty(cart.items));
      showToast("Добавлено в корзину");
    } catch (err) {
      showToast(err.message || "Не удалось добавить", true);
    } finally {
      btn.disabled = false;
    }
  }

  paintShell();
}

function productCard(p) {
  const img = productImageAttrs(p);
  const hasPhoto = !String(img.src).includes("/brands/");
  const suffix = String(p.id || "").replace(/-/g, "").slice(-4).toLowerCase();
  return `
    <article
      class="product"
      data-testid="product-card"
      data-product-id="${p.id}"
      data-product-suffix="${suffix}"
      data-brand="${escapeAttr(p.brand || "")}"
    >
      <div class="product-media ${hasPhoto ? "has-photo" : ""}" data-testid="product-media">
        <img
          class="product-image"
          src="${escapeAttr(img.src)}"
          alt="${escapeAttr(img.alt)}"
          width="400"
          height="300"
          data-fallbacks='${escapeAttr(img["data-fallbacks"])}'
          data-testid="product-image"
          loading="lazy"
          decoding="async"
        />
      </div>
      <div class="product-top">
        <span class="brand-tag" data-testid="product-brand">${escapeHtml(p.brand || "—")}</span>
        <span class="stock" data-testid="product-stock" data-stock="${Number(p.stockQuantity ?? 0)}">остаток ${Number(p.stockQuantity ?? 0)}</span>
      </div>
      <h2 data-testid="product-name">${escapeHtml(p.name)}</h2>
      <p class="desc" data-testid="product-description">${escapeHtml(p.description || "")}</p>
      <div class="product-foot">
        <span class="price" data-testid="product-price" data-price-cents="${Number(p.priceCents ?? 0)}">${formatMoney(p.priceCents)}</span>
        <button
          type="button"
          class="btn btn-primary"
          data-add="${p.id}"
          data-testid="add-to-cart"
          data-product-id="${p.id}"
        >В корзину</button>
      </div>
    </article>
  `;
}

function sumQty(items) {
  return (items || []).reduce((n, i) => n + Number(i.quantity || 0), 0);
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
