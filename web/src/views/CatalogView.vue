<script setup>
import { computed, onMounted, ref } from "vue";
import AppShell from "../components/AppShell.vue";
import ProductImage from "../components/ProductImage.vue";
import { api, formatMoney } from "../lib/api";
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

const { session } = useSession();
const { showToast } = useToast();
const shell = ref(null);

const loading = ref(true);
const loadError = ref("");
const products = ref([]);
const brand = ref("all");
const priceFloor = ref(0);
const priceCeil = ref(0);
const priceMin = ref(0);
const priceMax = ref(0);
const addingId = ref("");

const priceStep = computed(() =>
  Math.max(100, Math.round((priceCeil.value - priceFloor.value) / 100) || 100)
);

const filtered = computed(() =>
  products.value.filter((p) => {
    const brandOk = brand.value === "all" || p.brand === brand.value;
    const price = Number(p.priceCents) || 0;
    return brandOk && price >= priceMin.value && price <= priceMax.value;
  })
);

const brandCounts = computed(() => {
  const map = {};
  for (const b of BRANDS) {
    if (b.id === "all") {
      map.all = products.value.filter((p) => {
        const price = Number(p.priceCents) || 0;
        return price >= priceMin.value && price <= priceMax.value;
      }).length;
      continue;
    }
    map[b.id] = products.value.filter((p) => {
      const price = Number(p.priceCents) || 0;
      return p.brand === b.id && price >= priceMin.value && price <= priceMax.value;
    }).length;
  }
  return map;
});

const summary = computed(() => {
  const bits = [`${filtered.value.length} товаров`];
  if (brand.value !== "all") bits.push(brand.value);
  if (priceMin.value > priceFloor.value || priceMax.value < priceCeil.value) {
    bits.push(`${formatMoney(priceMin.value)}–${formatMoney(priceMax.value)}`);
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

function resetPrice() {
  priceMin.value = priceFloor.value;
  priceMax.value = priceCeil.value;
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

onMounted(async () => {
  try {
    const list = await api.listProducts();
    products.value = list.products || [];
    const cents = products.value.map((p) => Number(p.priceCents) || 0);
    priceFloor.value = cents.length ? Math.min(...cents) : 0;
    priceCeil.value = cents.length ? Math.max(...cents) : 0;
    priceMin.value = priceFloor.value;
    priceMax.value = priceCeil.value;
  } catch (err) {
    loadError.value = err.message || "Ошибка загрузки";
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <AppShell ref="shell" active="catalog">
    <p v-if="loading" class="loading" data-testid="catalog-loading">Загрузка каталога…</p>
    <div v-else-if="loadError" class="alert alert-error" data-testid="catalog-error">
      {{ loadError }}
    </div>
    <div v-else class="catalog-layout" data-testid="catalog-layout" data-page="catalog">
      <aside class="filters-panel" data-testid="catalog-filters">
        <h2 class="filters-title" data-testid="filters-title">Фильтры</h2>

        <p class="filters-hint" data-testid="filter-brand-label">Бренд</p>
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
            <span class="filter-count" :data-testid="`filter-${b.id}-count`">
              {{ brandCounts[b.id] ?? 0 }}
            </span>
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
          <button
            type="button"
            class="btn btn-ghost filter-reset"
            data-testid="price-reset"
            @click="resetPrice"
          >
            Сбросить цену
          </button>
        </div>
      </aside>

      <section class="catalog-main" data-testid="catalog-main">
        <div class="page-head">
          <div>
            <h1 data-testid="catalog-title">Каталог</h1>
            <p data-testid="catalog-summary">{{ summary }}</p>
          </div>
        </div>
        <div
          class="grid"
          data-testid="product-grid"
          :data-count="filtered.length"
        >
          <article
            v-for="p in filtered"
            :key="p.id"
            class="product"
            data-testid="product-card"
            :data-product-id="p.id"
            :data-product-suffix="productIdSuffix(p.id)"
            :data-brand="p.brand || ''"
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
                :disabled="addingId === p.id"
                @click="addToCart(p.id)"
              >
                В корзину
              </button>
            </div>
          </article>
          <div v-if="!filtered.length" class="empty" data-testid="catalog-empty">
            Нет товаров для этих фильтров
          </div>
        </div>
      </section>
    </div>
  </AppShell>
</template>
