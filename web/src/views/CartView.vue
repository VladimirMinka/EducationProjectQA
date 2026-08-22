<script setup>
import { computed, onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import AppShell from "../components/AppShell.vue";
import ProductImage from "../components/ProductImage.vue";
import { api, formatMoney } from "../lib/api";
import { productIdSuffix } from "../lib/images";
import { useSession } from "../composables/useSession";
import { useToast } from "../composables/useToast";

const { session } = useSession();
const { showToast } = useToast();
const shell = ref(null);

const loading = ref(true);
const loadError = ref("");
const productsById = ref(new Map());
const cart = ref(null);
const promoCode = ref("");
const notice = ref("");
const busy = ref(false);

const items = computed(() => cart.value?.items || []);
const totalQty = computed(() =>
  items.value.reduce((n, i) => n + Number(i.quantity || 0), 0)
);

function productFor(item) {
  return (
    productsById.value.get(item.productId) || {
      id: item.productId,
      brand: "",
      name: item.productId,
    }
  );
}

function formatExpires(value) {
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString("ru-RU");
}

function syncBadge() {
  shell.value?.setCartCount(totalQty.value);
}

async function run(fn) {
  if (busy.value) return;
  busy.value = true;
  try {
    await fn();
  } catch (err) {
    showToast(err.message || "Ошибка", true);
  } finally {
    busy.value = false;
    syncBadge();
  }
}

async function removeItem(productId) {
  const s = session.value;
  cart.value = await api.removeItem(s.user.id, s.accessToken, productId);
  showToast("Товар удалён");
}

async function applyPromo() {
  const code = promoCode.value.trim();
  if (!code) {
    showToast("Введите промокод", true);
    return;
  }
  const s = session.value;
  cart.value = await api.applyPromocode(s.user.id, s.accessToken, code);
  promoCode.value = "";
  showToast(`Промокод ${cart.value.appliedPromocode || code} применён`);
}

async function clearPromo() {
  const s = session.value;
  cart.value = await api.clearPromocode(s.user.id, s.accessToken);
  showToast("Промокод снят");
}

async function clearCart() {
  const s = session.value;
  cart.value = await api.clearCart(s.user.id, s.accessToken);
  notice.value = "";
  showToast("Корзина очищена");
}

async function checkout() {
  const s = session.value;
  const res = await api.createOrder(s.user.id, s.accessToken);
  const orderId = res?.order?.id || "—";
  cart.value = await api.getCart(s.user.id, s.accessToken);
  notice.value = `Заказ создан: ${orderId}`;
  showToast("Заказ оформлен");
}

onMounted(async () => {
  const s = session.value;
  try {
    const [list, cartRes] = await Promise.all([
      api.listProducts(),
      api.getCart(s.user.id, s.accessToken),
    ]);
    productsById.value = new Map((list.products || []).map((p) => [p.id, p]));
    cart.value = cartRes;
  } catch (err) {
    loadError.value = err.message || "Ошибка загрузки";
  } finally {
    loading.value = false;
    syncBadge();
  }
});
</script>

<template>
  <AppShell ref="shell" active="cart">
    <p v-if="loading" class="loading" data-testid="cart-loading">Загрузка корзины…</p>
    <div v-else-if="loadError" class="alert alert-error" data-testid="cart-error">
      {{ loadError }}
    </div>
    <template v-else>
      <div class="page-head" data-testid="cart-header">
        <div>
          <h1 data-testid="cart-title">Корзина</h1>
          <p data-testid="cart-subtitle">
            {{ items.length ? `${totalQty} шт.` : "Пока пусто" }}
          </p>
        </div>
      </div>
      <div
        v-if="notice"
        class="alert alert-ok"
        data-testid="cart-notice"
        data-order-notice="true"
      >
        {{ notice }}
      </div>
      <div
        class="cart-layout"
        data-testid="cart-page"
        data-page="cart"
        :data-items-count="items.length"
      >
        <div class="cart-list" data-testid="cart-list">
          <article
            v-for="item in items"
            :key="item.productId"
            class="cart-item"
            data-testid="cart-item"
            :data-product-id="item.productId"
            :data-product-suffix="productIdSuffix(item.productId)"
            :data-quantity="Number(item.quantity || 0)"
          >
            <ProductImage :product="productFor(item)" thumb :alt="productFor(item).name" />
            <div>
              <h3 data-testid="cart-item-name">{{ productFor(item).name }}</h3>
              <div class="cart-meta" data-testid="cart-item-meta">
                <span data-testid="cart-item-brand">{{ productFor(item).brand || "" }}</span>
                · qty
                <span data-testid="cart-item-qty">{{ Number(item.quantity || 0) }}</span>
                <template v-if="productFor(item).priceCents != null">
                  ·
                  <span data-testid="cart-item-line-total">
                    {{
                      formatMoney(
                        Number(productFor(item).priceCents) * Number(item.quantity || 0)
                      )
                    }}
                  </span>
                </template>
              </div>
            </div>
            <button
              type="button"
              class="btn btn-ghost"
              data-testid="remove-item"
              :data-product-id="item.productId"
              :disabled="busy"
              @click="run(() => removeItem(item.productId))"
            >
              Удалить
            </button>
          </article>
          <div v-if="!items.length" class="empty" data-testid="cart-empty">
            Корзина пуста.
            <RouterLink :to="{ name: 'catalog' }" data-testid="cart-empty-catalog-link">
              Перейти в каталог
            </RouterLink>
          </div>
        </div>

        <aside class="panel summary" data-testid="cart-summary">
          <div class="summary-row" data-testid="cart-row-subtotal">
            <span>Сабтотал</span>
            <span
              data-testid="cart-subtotal"
              :data-cents="Number(cart.subtotalCents ?? 0)"
            >{{ formatMoney(cart.subtotalCents) }}</span>
          </div>
          <div class="summary-row" data-testid="cart-row-discount">
            <span>Скидка</span>
            <span
              data-testid="cart-discount"
              :data-cents="Number(cart.discountCents ?? 0)"
            >{{ formatMoney(cart.discountCents) }}</span>
          </div>
          <div class="summary-row total" data-testid="cart-row-total">
            <span>Итого</span>
            <span
              data-testid="cart-total"
              :data-cents="Number(cart.totalPriceCents ?? 0)"
            >{{ formatMoney(cart.totalPriceCents) }}</span>
          </div>
          <div data-testid="cart-tags">
            <span
              v-if="cart.appliedPromocode"
              class="tag"
              data-testid="applied-promo"
              :data-promo="cart.appliedPromocode"
            >
              Промо: {{ cart.appliedPromocode }}
            </span>
            <span
              v-if="cart.comboDiscountApplied"
              class="tag"
              data-testid="combo-flag"
              data-combo="true"
            >
              Combo 10%
            </span>
          </div>
          <p
            v-if="items.length && cart.expiresAt"
            class="hint"
            data-testid="cart-expires"
            :data-expires-at="cart.expiresAt"
          >
            Корзина истечёт: {{ formatExpires(cart.expiresAt) }}
          </p>
          <div class="promo-row" data-testid="promo-row">
            <input
              v-model="promoCode"
              type="text"
              placeholder="Промокод"
              data-testid="promo-input"
              :disabled="busy"
            />
            <button
              type="button"
              class="btn btn-ghost"
              data-testid="promo-apply"
              :disabled="busy || !items.length"
              @click="run(applyPromo)"
            >
              OK
            </button>
          </div>
          <button
            v-if="cart.appliedPromocode"
            type="button"
            class="btn btn-ghost"
            data-testid="promo-clear"
            :disabled="busy"
            @click="run(clearPromo)"
          >
            Снять промокод
          </button>
          <button
            type="button"
            class="btn btn-primary"
            data-testid="checkout-button"
            :disabled="busy || !items.length"
            @click="run(checkout)"
          >
            Оформить заказ
          </button>
          <button
            type="button"
            class="btn btn-danger"
            data-testid="clear-cart-button"
            :disabled="busy || !items.length"
            @click="run(clearCart)"
          >
            Очистить
          </button>
        </aside>
      </div>
    </template>
  </AppShell>
</template>
