<script setup>
import { computed, onMounted, ref, watch } from "vue";
import AppShell from "../components/AppShell.vue";
import ProductImage from "../components/ProductImage.vue";
import { api, formatMoney, invalidateProductsCache } from "../lib/api";
import { productIdSuffix } from "../lib/images";
import { useSession } from "../composables/useSession";
import { useToast } from "../composables/useToast";

const BRANDS = [
  { id: "all", label: "Все" },
  { id: "apple", label: "Apple" },
  { id: "samsung", label: "Samsung" },
  { id: "nvidia", label: "NVIDIA" },
  { id: "amd", label: "AMD" },
];

const SORTS = [
  { id: "", label: "По умолчанию" },
  { id: "price_asc", label: "Цена ↑" },
  { id: "price_desc", label: "Цена ↓" },
  { id: "name_asc", label: "Имя A–Z" },
  { id: "name_desc", label: "Имя Z–A" },
];

const { session } = useSession();
const { showToast } = useToast();
const shell = ref(null);

const loading = ref(true);
const loadingMore = ref(false);
const loadError = ref("");
const products = ref([]);
const categories = ref([]);
const totalCount = ref(0);
const nextPageToken = ref("");

const brand = ref("all");
const categoryId = ref("");
const query = ref("");
const sort = ref("");
const inStockOnly = ref(false);
const priceFloor = ref(0);
const priceCeil = ref(500000);
const priceMin = ref(0);
const priceMax = ref(500000);
const addingId = ref("");

let fetchTimer = 0;

const priceStep = computed(() =>
  Math.max(100, Math.round((priceCeil.value - priceFloor.value) / 100) || 100)
);

const summary = computed(() => {
  const bits = [`${products.value.length} из ${totalCount.value}`];
  if (query.value.trim()) bits.push(`«${query.value.trim()}»`);
  if (brand.value !== "all") bits.push(brand.value);
  if (categoryId.value) {
    const cat = categories.value.find((c) => c.id === categoryId.value);
    if (cat) bits.push(cat.name);
  }
  return bits.join(" · ");
});

const rangeLeft = computed(() => {
  if (priceCeil.value <= priceFloor.value) return 0;
  return ((priceMin.value - priceFloor.value) / (priceCeil.value - priceFloor.value)) * 100;
});

const rangeWidth = computed(() => {
  if (priceCeil.value <= priceFloor.value) return 100;
  const right =
    ((priceMax.value - priceFloor.value) / (priceCeil.value - priceFloor.value)) * 100;
  return Math.max(0, right - rangeLeft.value);
});

function syncPrice(which) {
  let min = Number(priceMin.value);
  let max = Number(priceMax.value);
  if (min > max) {
    if (which === "min") max = min;
    else min = max;
    priceMin.value = min;
    priceMax.value = max;
  }
}

function resetFilters() {
  brand.value = "all";
  categoryId.value = "";
  query.value = "";
  sort.value = "";
  inStockOnly.value = false;
  priceMin.value = priceFloor.value;
  priceMax.value = priceCeil.value;
}

function buildParams(pageToken = "") {
  const params = { page_size: 20 };
  if (pageToken) params.page_token = pageToken;
  if (query.value.trim()) params.q = query.value.trim();
  if (brand.value !== "all") params.brand = brand.value;
  if (categoryId.value) params.category_id = categoryId.value;
  if (priceMin.value > priceFloor.value) params.min_price_cents = priceMin.value;
  if (priceMax.value < priceCeil.value) params.max_price_cents = priceMax.value;
  if (inStockOnly.value) params.in_stock = true;
  if (sort.value) params.sort = sort.value;
  return params;
}

async function fetchProducts({ append = false } = {}) {
  if (append) loadingMore.value = true;
  else loading.value = true;
  try {
    const token = append ? nextPageToken.value : "";
    const list = await api.listProducts(buildParams(token));
    const batch = list.products || [];
    products.value = append ? products.value.concat(batch) : batch;
    nextPageToken.value = list.nextPageToken || "";
    totalCount.value = Number(list.totalCount ?? products.value.length);
    loadError.value = "";
  } catch (err) {
    if (!append) {
      loadError.value = err.message || "Ошибка загрузки";
      products.value = [];
    } else {
      showToast(err.message || "Не удалось загрузить ещё", true);
    }
  } finally {
    loading.value = false;
    loadingMore.value = false;
  }
}

function scheduleFetch() {
  window.clearTimeout(fetchTimer);
  fetchTimer = window.setTimeout(() => fetchProducts({ append: false }), 250);
}

async function addToCart(productId) {
  const s = session.value;
  if (!s) return;
  addingId.value = productId;
  try {
    const cart = await api.addItem(s.user.id, s.accessToken, productId, 1);
    const count = (cart.items || []).reduce((n, i) => n + Number(i.quantity || 0), 0);
    shell.value?.setCartCount(count);
    showToast("Добавлено в корзину");
  } catch (err) {
    showToast(err.message || "Не удалось добавить", true);
  } finally {
    addingId.value = "";
  }
}

watch([brand, categoryId, sort, inStockOnly, priceMin, priceMax], scheduleFetch);
watch(query, scheduleFetch);

onMounted(async () => {
  try {
    invalidateProductsCache();
    const [bounds, cats] = await Promise.all([
      api.listProducts({ page_size: 50, sort: "price_asc" }),
      api.listCategories(),
    ]);
    categories.value = cats.categories || [];
    const cents = (bounds.products || []).map((p) => Number(p.priceCents) || 0);
    // Approximate bounds from first page sorted by price; also fetch price_desc for ceil
    const high = await api.listProducts({ page_size: 1, sort: "price_desc" });
    const lowCents = cents.length ? Math.min(...cents) : 0;
    const highCents = Number(high.products?.[0]?.priceCents) || (cents.length ? Math.max(...cents) : 0);
    priceFloor.value = lowCents;
    priceCeil.value = highCents;
    priceMin.value = lowCents;
    priceMax.value = highCents;
    await fetchProducts({ append: false });
  } catch (err) {
    loadError.value = err.message || "Ошибка загрузки";
    loading.value = false;
  }
});
</script>

<template>
  <AppShell ref="shell" active="catalog">
    <p v-if="loading && !products.length" class="loading" data-testid="catalog-loading">
      Загрузка каталога…
    </p>
    <div v-else-if="loadError && !products.length" class="alert alert-error" data-testid="catalog-error">
      {{ loadError }}
    </div>
    <div v-else class="catalog-layout" data-testid="catalog-layout" data-page="catalog">
      <aside class="filters-panel" data-testid="catalog-filters">
        <h2 class="filters-title" data-testid="filters-title">Фильтры</h2>

        <label class="filters-hint" for="catalog-search" data-testid="filter-search-label">Поиск</label>
        <input
          id="catalog-search"
          v-model="query"
          type="search"
          class="field-input"
          placeholder="iPhone, RTX…"
          data-testid="catalog-search"
        />

        <p class="filters-hint filters-hint-spaced" data-testid="filter-category-label">Категория</p>
        <div class="filter-list" data-testid="category-filter-list" role="list">
          <button
            type="button"
            class="filter-item"
            :class="{ active: !categoryId }"
            data-testid="filter-category-all"
            data-filter-type="category"
            :data-active="!categoryId ? 'true' : 'false'"
            role="listitem"
            @click="categoryId = ''"
          >
            <span>Все</span>
          </button>
          <button
            v-for="c in categories"
            :key="c.id"
            type="button"
            class="filter-item"
            :class="{ active: categoryId === c.id }"
            :data-category-id="c.id"
            :data-testid="`filter-category-${c.slug}`"
            data-filter-type="category"
            :data-active="categoryId === c.id ? 'true' : 'false'"
            role="listitem"
            @click="categoryId = c.id"
          >
            <span>{{ c.name }}</span>
          </button>
        </div>

        <p class="filters-hint filters-hint-spaced" data-testid="filter-brand-label">Бренд</p>
        <div class="filter-list" data-testid="brand-filter-list" role="list">
          <button
            v-for="b in BRANDS"
            :key="b.id"
            type="button"
            class="filter-item"
            :class="{ active: brand === b.id }"
            :data-brand="b.id"
            :data-testid="`filter-${b.id}`"
            data-filter-type="brand"
            :data-active="brand === b.id ? 'true' : 'false'"
            role="listitem"
            @click="brand = b.id"
          >
            <span :data-testid="`filter-${b.id}-label`">{{ b.label }}</span>
          </button>
        </div>

        <p class="filters-hint filters-hint-spaced" data-testid="filter-price-label">Цена</p>
        <div class="price-filter" data-testid="price-filter">
          <div class="price-labels">
            <span data-testid="price-min-label">{{ formatMoney(priceMin) }}</span>
            <span data-testid="price-max-label">{{ formatMoney(priceMax) }}</span>
          </div>
          <div class="range-wrap" data-testid="price-range">
            <div
              class="range-track"
              data-testid="price-range-track"
              :style="{ left: rangeLeft + '%', width: rangeWidth + '%' }"
            />
            <input
              v-model.number="priceMin"
              type="range"
              class="range-input range-min"
              data-testid="price-min"
              :min="priceFloor"
              :max="priceCeil"
              :step="priceStep"
              aria-label="Минимальная цена"
              @input="syncPrice('min')"
            />
            <input
              v-model.number="priceMax"
              type="range"
              class="range-input range-max"
              data-testid="price-max"
              :min="priceFloor"
              :max="priceCeil"
              :step="priceStep"
              aria-label="Максимальная цена"
              @input="syncPrice('max')"
            />
          </div>
          <p class="price-bounds" data-testid="price-bounds">
            {{ formatMoney(priceFloor) }} — {{ formatMoney(priceCeil) }}
          </p>
        </div>

        <label class="filter-check" data-testid="filter-in-stock">
          <input v-model="inStockOnly" type="checkbox" data-testid="in-stock-checkbox" />
          Только в наличии
        </label>

        <p class="filters-hint filters-hint-spaced" data-testid="filter-sort-label">Сортировка</p>
        <select v-model="sort" class="field-input" data-testid="catalog-sort">
          <option v-for="s in SORTS" :key="s.id || 'default'" :value="s.id">{{ s.label }}</option>
        </select>

        <button
          type="button"
          class="btn btn-ghost filter-reset"
          data-testid="filters-reset"
          @click="resetFilters"
        >
          Сбросить фильтры
        </button>
      </aside>

      <section class="catalog-main" data-testid="catalog-main">
        <div class="page-head">
          <div>
            <h1 data-testid="catalog-title">Каталог</h1>
            <p data-testid="catalog-summary">{{ summary }}</p>
          </div>
        </div>
        <div class="grid" data-testid="product-grid" :data-count="products.length">
          <article
            v-for="p in products"
            :key="p.id"
            class="product"
            data-testid="product-card"
            :data-product-id="p.id"
            :data-product-suffix="productIdSuffix(p.id)"
            :data-brand="p.brand || ''"
            :data-category-id="p.categoryId || ''"
          >
            <ProductImage :product="p" />
            <div class="product-top">
              <span class="brand-tag" data-testid="product-brand">{{ p.brand || "—" }}</span>
              <span
                class="stock"
                data-testid="product-stock"
                :data-stock="Number(p.stockQuantity ?? 0)"
              >
                остаток {{ Number(p.stockQuantity ?? 0) }}
              </span>
            </div>
            <h2 data-testid="product-name">{{ p.name }}</h2>
            <p class="desc" data-testid="product-description">{{ p.description || "" }}</p>
            <div class="product-foot">
              <span
                class="price"
                data-testid="product-price"
                :data-price-cents="Number(p.priceCents ?? 0)"
              >
                {{ formatMoney(p.priceCents) }}
              </span>
              <button
                type="button"
                class="btn btn-primary"
                data-testid="add-to-cart"
                :data-product-id="p.id"
                :disabled="addingId === p.id || Number(p.stockQuantity ?? 0) <= 0"
                @click="addToCart(p.id)"
              >
                В корзину
              </button>
            </div>
          </article>
          <div v-if="!products.length" class="empty" data-testid="catalog-empty">
            Нет товаров для этих фильтров
          </div>
        </div>
        <div v-if="nextPageToken" class="catalog-more">
          <button
            type="button"
            class="btn btn-ghost"
            data-testid="catalog-load-more"
            :disabled="loadingMore"
            @click="fetchProducts({ append: true })"
          >
            {{ loadingMore ? "Загрузка…" : "Показать ещё" }}
          </button>
        </div>
      </section>
    </div>
  </AppShell>
</template>
