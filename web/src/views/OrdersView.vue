<script setup>
import { computed, onMounted, ref } from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";
import AppShell from "../components/AppShell.vue";
import ProductImage from "../components/ProductImage.vue";
import { api, formatMoney } from "../lib/api";
import { productIdSuffix } from "../lib/images";
import { useSession } from "../composables/useSession";
import { useToast } from "../composables/useToast";
import { formatOrderDate, orderStatusKey, orderStatusLabel, deliveryMethodLabel, formatDeliveryInfo } from "../lib/orders";

const route = useRoute();
const router = useRouter();
const { session } = useSession();
const { showToast } = useToast();

const loading = ref(true);
const loadError = ref("");
const productsById = ref(new Map());
const orders = ref([]);
const busyId = ref("");

const placedId = computed(() => String(route.query.placed || ""));

function productFor(item) {
  return (
    productsById.value.get(item.productId) || {
      id: item.productId,
      brand: "",
      name: item.productId,
      priceCents: item.priceCents,
    }
  );
}

async function refresh() {
  const s = session.value;
  const res = await api.listOrders(s.user.id, s.accessToken);
  orders.value = res.orders || [];
}

async function cancelOrder(orderId) {
  if (busyId.value) return;
  busyId.value = orderId;
  try {
    await api.cancelOrder(orderId, session.value.accessToken);
    await refresh();
    showToast("Заказ отменён");
  } catch (err) {
    showToast(err.message || "Не удалось отменить", true);
  } finally {
    busyId.value = "";
  }
}

function payOrder(orderId) {
  router.push({ name: "checkout", query: { order: orderId } });
}

onMounted(async () => {
  const s = session.value;
  try {
    const [list] = await Promise.all([api.listProducts(), refresh()]);
    productsById.value = new Map((list.products || []).map((p) => [p.id, p]));
  } catch (err) {
    loadError.value = err.message || "Ошибка загрузки";
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <AppShell active="orders">
    <p v-if="loading" class="loading" data-testid="orders-loading">Загрузка заказов…</p>
    <div v-else-if="loadError" class="alert alert-error" data-testid="orders-error">
      {{ loadError }}
    </div>
    <template v-else>
      <div class="page-head" data-testid="orders-header">
        <div>
          <h1 data-testid="orders-title">Заказы</h1>
          <p data-testid="orders-subtitle">
            {{ orders.length ? `${orders.length} шт.` : "Пока пусто" }}
          </p>
        </div>
      </div>
      <div
        v-if="placedId"
        class="alert alert-ok"
        data-testid="orders-placed-notice"
        data-order-notice="true"
        :data-order-id="placedId"
      >
        Заказ оформлен: {{ placedId }}
      </div>
      <div class="orders-list" data-testid="orders-page" data-page="orders" :data-count="orders.length">
        <article
          v-for="order in orders"
          :key="order.id"
          class="order-card"
          data-testid="order-card"
          :data-order-id="order.id"
          :data-status="orderStatusKey(order.status)"
        >
          <div class="order-head">
            <div>
              <h2 data-testid="order-id">Заказ {{ order.id.slice(0, 8) }}</h2>
              <p class="cart-meta" data-testid="order-date">{{ formatOrderDate(order.createdAt) }}</p>
            </div>
            <span
              class="status-pill"
              data-testid="order-status"
              :data-status="orderStatusKey(order.status)"
            >
              {{ orderStatusLabel(order.status) }}
            </span>
          </div>
          <div class="order-items" data-testid="order-items">
            <div
              v-for="item in order.items || []"
              :key="item.productId"
              class="order-line"
              data-testid="order-item"
              :data-product-id="item.productId"
              :data-product-suffix="productIdSuffix(item.productId)"
            >
              <ProductImage :product="productFor(item)" thumb :alt="productFor(item).name" />
              <div>
                <p data-testid="order-item-name">{{ productFor(item).name }}</p>
                <p class="cart-meta">
                  qty <span data-testid="order-item-qty">{{ Number(item.quantity || 0) }}</span>
                  ·
                  <span data-testid="order-item-line-total">
                    {{ formatMoney(Number(item.priceCents || 0) * Number(item.quantity || 0)) }}
                  </span>
                </p>
              </div>
            </div>
          </div>
          <p class="cart-meta" data-testid="order-delivery">
            <span data-testid="order-delivery-method">{{ deliveryMethodLabel(order.deliveryMethod) }}</span>
            ·
            <span data-testid="order-delivery-info">{{ formatDeliveryInfo(order) }}</span>
            · доставка
            <span data-testid="order-delivery-fee" :data-cents="Number(order.deliveryFeeCents ?? 0)">
              {{ formatMoney(order.deliveryFeeCents) }}
            </span>
          </p>
          <div class="order-foot">
            <span class="price" data-testid="order-total" :data-cents="Number(order.totalAmountCents ?? 0)">
              {{ formatMoney(order.totalAmountCents) }}
            </span>
            <div class="order-actions">
              <button
                v-if="orderStatusKey(order.status) === 'CREATED'"
                type="button"
                class="btn btn-primary"
                data-testid="pay-order"
                :data-order-id="order.id"
                :disabled="busyId === order.id"
                @click="payOrder(order.id)"
              >
                Оплатить
              </button>
              <button
                v-if="orderStatusKey(order.status) === 'CREATED'"
                type="button"
                class="btn btn-ghost"
                data-testid="cancel-order"
                :data-order-id="order.id"
                :disabled="busyId === order.id"
                @click="cancelOrder(order.id)"
              >
                Отменить
              </button>
            </div>
          </div>
        </article>
        <div v-if="!orders.length" class="empty" data-testid="orders-empty">
          Заказов ещё нет.
          <RouterLink :to="{ name: 'catalog' }" data-testid="orders-empty-catalog-link">
            Перейти в каталог
          </RouterLink>
        </div>
      </div>
    </template>
  </AppShell>
</template>
