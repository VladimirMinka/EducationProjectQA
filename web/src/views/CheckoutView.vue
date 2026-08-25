<script setup>
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import AppShell from "../components/AppShell.vue";
import ProductImage from "../components/ProductImage.vue";
import { api, formatMoney } from "../lib/api";
import { productIdSuffix } from "../lib/images";
import { useSession } from "../composables/useSession";
import { useToast } from "../composables/useToast";
import {
  PAY_SCHEMES,
  brandLabel,
  cvcLengthForBrand,
  detectBrand,
  digitsOnly,
  formatCardNumber,
  formatExpiry,
  isExpiryValid,
  luhnCheck,
  maskCardNumber,
  secureProgram,
} from "../lib/luhn";
import { orderStatusKey } from "../lib/orders";

const route = useRoute();
const router = useRouter();
const { session } = useSession();
const { showToast } = useToast();

const loading = ref(true);
const loadError = ref("");
const productsById = ref(new Map());
const cart = ref(null);
const existingOrder = ref(null);
const busy = ref(false);

const pan = ref("");
const expiry = ref("");
const cvc = ref("");
const holder = ref("");
const touched = ref({ pan: false, expiry: false, cvc: false, holder: false });

const step = ref("form");
const otp = ref("");
const otpExpected = ref("");
const otpError = ref("");
const otpAttempts = ref(0);
const phoneTail = ref("47");
const showSms = ref(false);
let connectTimer = 0;
let smsTimer = 0;

const items = computed(() => {
  if (existingOrder.value) {
    return existingOrder.value.items || [];
  }
  return cart.value?.items || [];
});

const totalCents = computed(() => {
  if (existingOrder.value) return Number(existingOrder.value.totalAmountCents ?? 0);
  return Number(cart.value?.totalPriceCents ?? 0);
});

const subtotalCents = computed(() => {
  if (existingOrder.value) return totalCents.value;
  return Number(cart.value?.subtotalCents ?? 0);
});

const discountCents = computed(() => {
  if (existingOrder.value) return 0;
  return Number(cart.value?.discountCents ?? 0);
});

const brand = computed(() => detectBrand(pan.value));
const panValid = computed(() => luhnCheck(pan.value));
const expiryValid = computed(() => isExpiryValid(expiry.value));
const cvcValid = computed(() => {
  const need = cvcLengthForBrand(brand.value);
  return digitsOnly(cvc.value).length === need;
});
const holderValid = computed(() => holder.value.trim().length >= 2);
const formValid = computed(
  () => panValid.value && expiryValid.value && cvcValid.value && holderValid.value
);

const panError = computed(() => {
  if (!touched.value.pan) return "";
  const digits = digitsOnly(pan.value);
  if (digits.length < 13) return "Введите номер карты";
  if (!panValid.value) return "Неверный номер карты";
  return "";
});

const expiryError = computed(() => {
  if (!touched.value.expiry) return "";
  if (!expiryValid.value) return "Неверный срок";
  return "";
});

const cvcError = computed(() => {
  if (!touched.value.cvc) return "";
  if (!cvcValid.value) return "Неверный CVC";
  return "";
});

const holderError = computed(() => {
  if (!touched.value.holder) return "";
  if (!holderValid.value) return "Укажите имя на карте";
  return "";
});

const maskedPan = computed(() => maskCardNumber(pan.value) || "•••• •••• •••• ••••");
const last4 = computed(() => {
  const digits = digitsOnly(pan.value);
  return digits.length >= 4 ? digits.slice(-4) : "••••";
});
const acs = computed(() => secureProgram(brand.value));

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

function lineTotal(item) {
  const product = productFor(item);
  const price = Number(item.priceCents ?? product.priceCents ?? 0);
  return price * Number(item.quantity || 0);
}

function onPanInput(event) {
  pan.value = formatCardNumber(event.target.value);
}

function onExpiryInput(event) {
  expiry.value = formatExpiry(event.target.value);
}

function onCvcInput(event) {
  cvc.value = digitsOnly(event.target.value).slice(0, cvcLengthForBrand(brand.value));
}

function clearChallengeTimers() {
  window.clearTimeout(connectTimer);
  window.clearTimeout(smsTimer);
  connectTimer = 0;
  smsTimer = 0;
}

function startPay() {
  touched.value = { pan: true, expiry: true, cvc: true, holder: true };
  if (!formValid.value) {
    showToast("Проверьте данные карты", true);
    return;
  }
  otpExpected.value = String(Math.floor(100000 + Math.random() * 900000));
  phoneTail.value = String(Math.floor(10 + Math.random() * 90));
  otp.value = "";
  otpError.value = "";
  otpAttempts.value = 0;
  showSms.value = false;
  step.value = "connecting";
  clearChallengeTimers();
  connectTimer = window.setTimeout(() => {
    step.value = "threeds";
    smsTimer = window.setTimeout(() => {
      showSms.value = true;
    }, 650);
  }, 1300);
}

function cancelThreeds() {
  clearChallengeTimers();
  step.value = "form";
  otpExpected.value = "";
  otp.value = "";
  otpError.value = "";
  showSms.value = false;
}

async function confirmThreeds() {
  if (digitsOnly(otp.value) !== otpExpected.value) {
    otpAttempts.value += 1;
    if (otpAttempts.value >= 3) {
      showToast("Оплата отклонена банком", true);
      cancelThreeds();
      return;
    }
    otpError.value = `Неверный код · осталось попыток: ${3 - otpAttempts.value}`;
    return;
  }
  await submitPaidOrder();
}

async function submitPaidOrder() {
  if (busy.value) return;
  busy.value = true;
  step.value = "paying";
  const s = session.value;
  let orderId = existingOrder.value?.id;
  let createdNow = false;
  try {
    if (!orderId) {
      const res = await api.createOrder(s.user.id, s.accessToken);
      orderId = res?.order?.id;
      createdNow = true;
    }
    try {
      await api.updateOrderStatus(
        orderId,
        s.accessToken,
        "ORDER_STATUS_CREATED",
        "ORDER_STATUS_PAID"
      );
    } catch (err) {
      if (createdNow && orderId) {
        showToast("Заказ создан, но оплата не прошла — доплатите в заказах", true);
        router.push({ name: "orders", query: { placed: orderId } });
        return;
      }
      throw err;
    }
    showToast("Оплата прошла");
    router.push({ name: "orders", query: { placed: orderId } });
  } catch (err) {
    showToast(err.message || "Не удалось оплатить", true);
    step.value = "form";
  } finally {
    busy.value = false;
  }
}

onMounted(async () => {
  const s = session.value;
  holder.value = s?.user?.name || "";
  try {
    const list = await api.listProducts();
    productsById.value = new Map((list.products || []).map((p) => [p.id, p]));
    const orderId = String(route.query.order || "");
    if (orderId) {
      const res = await api.getOrder(orderId, s.accessToken);
      const order = res.order;
      if (orderStatusKey(order?.status) !== "CREATED") {
        loadError.value = "Этот заказ уже нельзя оплатить с карты";
      } else {
        existingOrder.value = order;
      }
    } else {
      const cartRes = await api.getCart(s.user.id, s.accessToken);
      cart.value = cartRes;
      if (!(cartRes.items || []).length) {
        router.replace({ name: "cart" });
        return;
      }
    }
  } catch (err) {
    loadError.value = err.message || "Ошибка загрузки";
  } finally {
    loading.value = false;
  }
});

onUnmounted(clearChallengeTimers);
</script>

<template>
  <AppShell active="checkout">
    <p v-if="loading" class="loading" data-testid="checkout-loading">Готовим оплату…</p>
    <div v-else-if="loadError" class="alert alert-error" data-testid="checkout-error">
      {{ loadError }}
    </div>
    <template v-else>
      <div class="page-head" data-testid="checkout-header">
        <div>
          <h1 data-testid="checkout-title">Оплата</h1>
          <p data-testid="checkout-subtitle">Банковская карта</p>
        </div>
      </div>

      <div
        class="checkout-layout"
        data-testid="checkout-page"
        data-page="checkout"
        :data-step="step"
        :data-luhn="panValid ? 'valid' : 'invalid'"
        :data-brand="brand || ''"
      >
        <div class="checkout-items" data-testid="checkout-items">
          <article
            v-for="item in items"
            :key="item.productId"
            class="cart-item"
            data-testid="checkout-item"
            :data-product-id="item.productId"
            :data-product-suffix="productIdSuffix(item.productId)"
          >
            <ProductImage :product="productFor(item)" thumb :alt="productFor(item).name" />
            <div>
              <h3 data-testid="checkout-item-name">{{ productFor(item).name }}</h3>
              <div class="cart-meta">
                qty
                <span data-testid="checkout-item-qty">{{ Number(item.quantity || 0) }}</span>
                ·
                <span data-testid="checkout-item-line-total">{{ formatMoney(lineTotal(item)) }}</span>
              </div>
            </div>
          </article>
        </div>

        <aside class="panel summary checkout-pay" data-testid="checkout-pay">
          <div class="pay-visual" data-testid="card-visual" :data-brand="brand || 'unknown'">
            <div class="pay-visual-top">
              <span class="pay-chip" aria-hidden="true" />
              <div class="pay-schemes" data-testid="card-schemes">
                <img
                  v-for="scheme in PAY_SCHEMES"
                  :key="scheme.id"
                  class="pay-scheme"
                  :class="{ active: brand === scheme.id }"
                  :src="scheme.src"
                  :alt="scheme.label"
                  :title="scheme.label"
                  :data-scheme="scheme.id"
                  :data-testid="`scheme-${scheme.id}`"
                  :data-active="brand === scheme.id ? 'true' : 'false'"
                />
                <span class="pay-brand" data-testid="card-brand">{{ brandLabel(brand) || "CARD" }}</span>
              </div>
            </div>
            <p class="pay-pan" data-testid="card-visual-number">{{ maskedPan }}</p>
            <div class="pay-visual-foot">
              <span data-testid="card-visual-holder">{{ holder.trim() || "NAME SURNAME" }}</span>
              <span data-testid="card-visual-expiry">{{ expiry || "MM/YY" }}</span>
            </div>
          </div>

          <form class="card-form" data-testid="card-form" @submit.prevent="startPay">
            <div class="field field-pan">
              <label for="card-number">Номер карты</label>
              <input
                id="card-number"
                :value="formatCardNumber(pan)"
                type="text"
                inputmode="numeric"
                autocomplete="cc-number"
                placeholder="ACCT-000006"
                data-testid="card-number"
                :data-luhn="panValid ? 'valid' : 'invalid'"
                :class="{ 'input-invalid': panError }"
                maxlength="23"
                @input="onPanInput"
                @blur="touched.pan = true"
              />
              <p v-if="panError" class="field-error" data-testid="card-number-error">{{ panError }}</p>
            </div>
            <div class="card-form-row">
              <div class="field">
                <label for="card-expiry">Срок</label>
                <input
                  id="card-expiry"
                  :value="expiry"
                  type="text"
                  inputmode="numeric"
                  autocomplete="cc-exp"
                  placeholder="MM/YY"
                  data-testid="card-expiry"
                  :class="{ 'input-invalid': expiryError }"
                  maxlength="5"
                  @input="onExpiryInput"
                  @blur="touched.expiry = true"
                />
                <p class="field-error" data-testid="card-expiry-error">{{ expiryError }}</p>
              </div>
              <div class="field">
                <label for="card-cvc">CVC</label>
                <input
                  id="card-cvc"
                  :value="cvc"
                  type="text"
                  inputmode="numeric"
                  autocomplete="off"
                  placeholder="•••"
                  data-testid="card-cvc"
                  class="cvc-input"
                  :class="{ 'input-invalid': cvcError }"
                  :maxlength="cvcLengthForBrand(brand)"
                  @input="onCvcInput"
                  @blur="touched.cvc = true"
                />
                <p class="field-error" data-testid="card-cvc-error">{{ cvcError }}</p>
              </div>
            </div>
            <div class="field">
              <label for="card-holder">Имя на карте</label>
              <input
                id="card-holder"
                v-model="holder"
                type="text"
                autocomplete="cc-name"
                placeholder="IVAN IVANOV"
                data-testid="card-holder"
                :class="{ 'input-invalid': holderError }"
                @blur="touched.holder = true"
              />
              <p v-if="holderError" class="field-error" data-testid="card-holder-error">{{ holderError }}</p>
            </div>

            <div class="summary-row" data-testid="checkout-row-subtotal">
              <span>Сабтотал</span>
              <span data-testid="checkout-subtotal">{{ formatMoney(subtotalCents) }}</span>
            </div>
            <div class="summary-row" data-testid="checkout-row-discount">
              <span>Скидка</span>
              <span data-testid="checkout-discount">{{ formatMoney(discountCents) }}</span>
            </div>
            <div class="summary-row total" data-testid="checkout-row-total">
              <span>К оплате</span>
              <span data-testid="checkout-total" :data-cents="totalCents">
                {{ formatMoney(totalCents) }}
              </span>
            </div>

            <button
              type="submit"
              class="btn btn-primary"
              data-testid="pay-button"
              :disabled="busy || step !== 'form'"
            >
              Оплатить {{ formatMoney(totalCents) }}
            </button>
          </form>
        </aside>
      </div>
    </template>

    <div
      v-if="step !== 'form'"
      class="threeds-backdrop"
      data-testid="threeds-modal"
      :data-state="step"
    >
      <div v-if="step === 'connecting'" class="acs-wait" data-testid="threeds-connecting">
        <span class="acs-spinner" aria-hidden="true" />
        <p>Связываемся с банком…</p>
        <span>Visa Secure · Mastercard Identity Check</span>
      </div>

      <div
        v-else
        class="acs-shell"
        role="dialog"
        aria-modal="true"
        aria-labelledby="threeds-title"
        :data-brand="brand || 'unknown'"
      >
        <div class="acs-chrome">
          <span class="acs-lock" aria-hidden="true" />
          <span class="acs-url" data-testid="threeds-url">https://{{ acs.host }}</span>
        </div>
        <div class="acs-body">
          <div class="acs-brand-row">
            <span class="acs-mark" :data-brand="brand || 'visa'" aria-hidden="true" />
            <span class="acs-program" data-testid="threeds-program">{{ acs.name }}</span>
          </div>
          <h2 id="threeds-title" data-testid="threeds-title">Подтверждение оплаты</h2>
          <dl class="acs-details">
            <div>
              <dt>Магазин</dt>
              <dd>STORE</dd>
            </div>
            <div>
              <dt>Сумма</dt>
              <dd data-testid="threeds-amount">{{ formatMoney(totalCents) }}</dd>
            </div>
            <div>
              <dt>Карта</dt>
              <dd data-testid="threeds-card">•••• {{ last4 }}</dd>
            </div>
          </dl>
          <p class="acs-copy">
            Код отправлен на номер, привязанный к карте •• {{ phoneTail }}.
            Никому не сообщайте его.
          </p>
          <div class="field">
            <label for="threeds-otp">Код из SMS</label>
            <input
              id="threeds-otp"
              :value="otp"
              type="text"
              inputmode="numeric"
              maxlength="6"
              placeholder="••••••"
              data-testid="threeds-otp"
              :disabled="step === 'paying'"
              autocomplete="one-time-code"
              @input="otp = digitsOnly($event.target.value).slice(0, 6)"
              @keyup.enter="confirmThreeds"
            />
            <p v-if="otpError" class="field-error" data-testid="threeds-otp-error">{{ otpError }}</p>
          </div>
          <div class="threeds-actions">
            <button
              type="button"
              class="btn btn-ghost acs-cancel"
              data-testid="threeds-cancel"
              :disabled="step === 'paying'"
              @click="cancelThreeds"
            >
              Отмена
            </button>
            <button
              type="button"
              class="btn btn-primary acs-confirm"
              data-testid="threeds-confirm"
              :disabled="step === 'paying' || digitsOnly(otp).length !== 6"
              @click="confirmThreeds"
            >
              {{ step === "paying" ? "Обработка…" : "Подтвердить" }}
            </button>
          </div>
          <p class="acs-foot">Защищено {{ acs.name }}</p>
        </div>
      </div>
    </div>

    <aside
      v-if="showSms && step === 'threeds'"
      class="sms-toast"
      data-testid="threeds-otp-hint"
      :data-otp="otpExpected"
    >
      <p class="sms-from">Банк</p>
      <p class="sms-body">
        Код: <strong>{{ otpExpected }}</strong>
        для оплаты в STORE. Никому не сообщайте.
      </p>
    </aside>
  </AppShell>
</template>
