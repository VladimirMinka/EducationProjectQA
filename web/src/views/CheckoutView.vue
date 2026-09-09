<script setup>
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import AppShell from "../components/AppShell.vue";
import ProductImage from "../components/ProductImage.vue";
import {
  api,
  deliveryFeeFor,
  formatMoney,
  FREE_DELIVERY_SUBTOTAL_CENTS,
} from "../lib/api";
import { productIdSuffix } from "../lib/images";
import { useSession } from "../composables/useSession";
import { useToast } from "../composables/useToast";
import {
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

const phase = ref("delivery"); // delivery | payment
const deliveryMethod = ref("COURIER");
const addresses = ref([]);
const pickupPoints = ref([]);
const selectedAddressId = ref("");
const selectedPickupId = ref("");
const showAddressForm = ref(false);
const addressForm = ref({
  title: "Дом",
  city: "Москва",
  street: "",
  building: "",
  apartment: "",
  postalCode: "",
  recipientName: "",
  phone: "",
  isDefault: true,
});

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

const merchandiseSubtotal = computed(() => {
  if (existingOrder.value) {
    const fee = Number(existingOrder.value.deliveryFeeCents ?? 0);
    return Number(existingOrder.value.totalAmountCents ?? 0) - fee;
  }
  return Number(cart.value?.subtotalCents ?? 0);
});

const discountCents = computed(() => {
  if (existingOrder.value) return 0;
  return Number(cart.value?.discountCents ?? 0);
});

const deliveryFeeCents = computed(() => {
  if (existingOrder.value) return Number(existingOrder.value.deliveryFeeCents ?? 0);
  return deliveryFeeFor(deliveryMethod.value, merchandiseSubtotal.value);
});

const totalCents = computed(() => {
  if (existingOrder.value) return Number(existingOrder.value.totalAmountCents ?? 0);
  const goods = Number(cart.value?.totalPriceCents ?? 0);
  return goods + deliveryFeeCents.value;
});

const deliveryReady = computed(() => {
  if (existingOrder.value) return true;
  if (deliveryMethod.value === "COURIER") return Boolean(selectedAddressId.value);
  if (deliveryMethod.value === "PICKUP") return Boolean(selectedPickupId.value);
  return false;
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

function goToPayment() {
  if (!deliveryReady.value) {
    showToast("Выберите способ получения", true);
    return;
  }
  phase.value = "payment";
}

async function saveAddress() {
  const s = session.value;
  const f = addressForm.value;
  if (!f.city || !f.street || !f.building || !f.recipientName || !f.phone) {
    showToast("Заполните обязательные поля адреса", true);
    return;
  }
  busy.value = true;
  try {
    const res = await api.createAddress(s.user.id, s.accessToken, {
      title: f.title,
      city: f.city,
      street: f.street,
      building: f.building,
      apartment: f.apartment,
      postal_code: f.postalCode,
      recipient_name: f.recipientName,
      phone: f.phone,
      is_default: f.isDefault,
    });
    addresses.value = [...addresses.value, res.address];
    selectedAddressId.value = res.address.id;
    showAddressForm.value = false;
    showToast("Адрес сохранён");
  } catch (err) {
    showToast(err.message || "Не удалось сохранить адрес", true);
  } finally {
    busy.value = false;
  }
}

function startPay() {
  touched.value = { pan: true, expiry: true, cvc: true, holder: true };
  if (!formValid.value) {
    showToast("Проверьте данные карты", true);
    return;
  }
  if (!deliveryReady.value) {
    showToast("Выберите доставку", true);
    phase.value = "delivery";
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
      const method =
        deliveryMethod.value === "PICKUP"
          ? "DELIVERY_METHOD_PICKUP"
          : "DELIVERY_METHOD_COURIER";
      const res = await api.createOrder(s.user.id, s.accessToken, {
        deliveryMethod: method,
        addressId: deliveryMethod.value === "COURIER" ? selectedAddressId.value : "",
        pickupPointId: deliveryMethod.value === "PICKUP" ? selectedPickupId.value : "",
      });
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
  addressForm.value.recipientName = s?.user?.name || "";
  try {
    const list = await api.listProducts({ page_size: 50 });
    productsById.value = new Map((list.products || []).map((p) => [p.id, p]));
    const orderId = String(route.query.order || "");
    if (orderId) {
      const res = await api.getOrder(orderId, s.accessToken);
      const order = res.order;
      if (orderStatusKey(order?.status) !== "CREATED") {
        loadError.value = "Этот заказ уже нельзя оплатить с карты";
      } else {
        existingOrder.value = order;
        phase.value = "payment";
      }
    } else {
      const [cartRes, addrRes, pickupRes] = await Promise.all([
        api.getCart(s.user.id, s.accessToken),
        api.listAddresses(s.user.id, s.accessToken),
        api.listPickupPoints(),
      ]);
      cart.value = cartRes;
      if (!(cartRes.items || []).length) {
        router.replace({ name: "cart" });
        return;
      }
      addresses.value = addrRes.addresses || [];
      pickupPoints.value = pickupRes.pickupPoints || [];
      const def = addresses.value.find((a) => a.isDefault) || addresses.value[0];
      if (def) selectedAddressId.value = def.id;
      if (pickupPoints.value[0]) selectedPickupId.value = pickupPoints.value[0].id;
      showAddressForm.value = addresses.value.length === 0;
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
          <h1 data-testid="checkout-title">
            {{ phase === "delivery" ? "Доставка" : "Оплата" }}
          </h1>
          <p data-testid="checkout-subtitle">
            {{ phase === "delivery" ? "Способ получения" : "Банковская карта" }}
          </p>
        </div>
      </div>

      <div
        class="checkout-layout"
        data-testid="checkout-page"
        data-page="checkout"
        :data-phase="phase"
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

          <section
            v-if="phase === 'delivery'"
            class="panel delivery-panel"
            data-testid="checkout-delivery"
          >
            <h2 data-testid="checkout-delivery-title">Как получить заказ</h2>
            <div class="delivery-methods" data-testid="delivery-method-list">
              <button
                type="button"
                class="filter-item"
                :class="{ active: deliveryMethod === 'COURIER' }"
                data-testid="delivery-method-courier"
                :data-active="deliveryMethod === 'COURIER' ? 'true' : 'false'"
                @click="deliveryMethod = 'COURIER'"
              >
                Курьер
              </button>
              <button
                type="button"
                class="filter-item"
                :class="{ active: deliveryMethod === 'PICKUP' }"
                data-testid="delivery-method-pickup"
                :data-active="deliveryMethod === 'PICKUP' ? 'true' : 'false'"
                @click="deliveryMethod = 'PICKUP'"
              >
                Самовывоз
              </button>
            </div>

            <div v-if="deliveryMethod === 'COURIER'" data-testid="courier-block">
              <p class="filters-hint">Адрес доставки</p>
              <div class="filter-list" data-testid="address-list">
                <button
                  v-for="a in addresses"
                  :key="a.id"
                  type="button"
                  class="filter-item"
                  :class="{ active: selectedAddressId === a.id }"
                  data-testid="address-option"
                  :data-address-id="a.id"
                  :data-active="selectedAddressId === a.id ? 'true' : 'false'"
                  @click="selectedAddressId = a.id"
                >
                  <span>
                    {{ a.city }}, {{ a.street }}, {{ a.building }}
                    <span v-if="a.isDefault"> · default</span>
                  </span>
                </button>
              </div>
              <button
                type="button"
                class="btn btn-ghost"
                data-testid="address-add-toggle"
                @click="showAddressForm = !showAddressForm"
              >
                {{ showAddressForm ? "Скрыть форму" : "Новый адрес" }}
              </button>
              <div v-if="showAddressForm" class="address-form" data-testid="address-form">
                <div class="field">
                  <label>Город</label>
                  <input v-model="addressForm.city" data-testid="address-city" />
                </div>
                <div class="field">
                  <label>Улица</label>
                  <input v-model="addressForm.street" data-testid="address-street" />
                </div>
                <div class="field">
                  <label>Дом</label>
                  <input v-model="addressForm.building" data-testid="address-building" />
                </div>
                <div class="field">
                  <label>Квартира</label>
                  <input v-model="addressForm.apartment" data-testid="address-apartment" />
                </div>
                <div class="field">
                  <label>Получатель</label>
                  <input v-model="addressForm.recipientName" data-testid="address-recipient" />
                </div>
                <div class="field">
                  <label>Телефон</label>
                  <input v-model="addressForm.phone" data-testid="address-phone" />
                </div>
                <button
                  type="button"
                  class="btn btn-primary"
                  data-testid="address-save"
                  :disabled="busy"
                  @click="saveAddress"
                >
                  Сохранить адрес
                </button>
              </div>
            </div>

            <div v-else data-testid="pickup-block">
              <p class="filters-hint">Пункт выдачи</p>
              <div class="filter-list" data-testid="pickup-list">
                <button
                  v-for="p in pickupPoints"
                  :key="p.id"
                  type="button"
                  class="filter-item"
                  :class="{ active: selectedPickupId === p.id }"
                  data-testid="pickup-option"
                  :data-pickup-id="p.id"
                  :data-active="selectedPickupId === p.id ? 'true' : 'false'"
                  @click="selectedPickupId = p.id"
                >
                  <span>{{ p.code }} · {{ p.city }}, {{ p.address }}</span>
                </button>
              </div>
            </div>
          </section>
        </div>

        <aside class="panel summary checkout-pay" data-testid="checkout-pay">
          <template v-if="phase === 'delivery'">
            <div class="summary-row" data-testid="checkout-row-subtotal">
              <span>Товары</span>
              <span data-testid="checkout-subtotal">{{ formatMoney(merchandiseSubtotal) }}</span>
            </div>
            <div class="summary-row" data-testid="checkout-row-delivery">
              <span>Доставка</span>
              <span data-testid="checkout-delivery-fee" :data-cents="deliveryFeeCents">
                {{ formatMoney(deliveryFeeCents) }}
              </span>
            </div>
            <p
              v-if="deliveryMethod === 'COURIER' && merchandiseSubtotal < FREE_DELIVERY_SUBTOTAL_CENTS"
              class="filters-hint"
              data-testid="free-delivery-hint"
            >
              Бесплатно от {{ formatMoney(FREE_DELIVERY_SUBTOTAL_CENTS) }}
            </p>
            <div class="summary-row total" data-testid="checkout-row-total">
              <span>Итого</span>
              <span data-testid="checkout-total" :data-cents="totalCents">
                {{ formatMoney(totalCents) }}
              </span>
            </div>
            <button
              type="button"
              class="btn btn-primary"
              data-testid="checkout-to-payment"
              :disabled="!deliveryReady"
              @click="goToPayment"
            >
              К оплате
            </button>
          </template>

          <form v-else data-testid="card-form" @submit.prevent="startPay">
            <button
              v-if="!existingOrder"
              type="button"
              class="btn btn-ghost"
              data-testid="checkout-back-delivery"
              @click="phase = 'delivery'"
            >
              ← Доставка
            </button>
            <div class="pay-visual" data-testid="card-visual" :data-brand="brand || 'unknown'">
              <div class="pay-visual-top">
                <span class="pay-chip" aria-hidden="true" />
                <span data-testid="card-brand-label">{{ brandLabel(brand) }}</span>
              </div>
              <p class="pay-pan" data-testid="card-pan-mask">{{ maskedPan }}</p>
              <div class="pay-visual-bottom">
                <span data-testid="card-holder-mask">{{ holder || "NAME" }}</span>
                <span data-testid="card-expiry-mask">{{ expiry || "MM/YY" }}</span>
              </div>
            </div>

            <div class="field">
              <label for="card-pan">Номер карты</label>
              <input
                id="card-pan"
                :value="pan"
                type="text"
                inputmode="numeric"
                autocomplete="cc-number"
                placeholder="4111 1111 1111 1111"
                data-testid="card-pan"
                :class="{ 'input-invalid': panError }"
                @input="onPanInput"
                @blur="touched.pan = true"
              />
              <p v-if="panError" class="field-error" data-testid="card-pan-error">{{ panError }}</p>
            </div>
            <div class="field-row">
              <div class="field">
                <label for="card-expiry">Срок</label>
                <input
                  id="card-expiry"
                  :value="expiry"
                  type="text"
                  inputmode="numeric"
                  autocomplete="cc-exp"
                  placeholder="12/28"
                  data-testid="card-expiry"
                  :class="{ 'input-invalid': expiryError }"
                  @input="onExpiryInput"
                  @blur="touched.expiry = true"
                />
                <p v-if="expiryError" class="field-error" data-testid="card-expiry-error">
                  {{ expiryError }}
                </p>
              </div>
              <div class="field">
                <label for="card-cvc">CVC</label>
                <input
                  id="card-cvc"
                  :value="cvc"
                  type="password"
                  inputmode="numeric"
                  autocomplete="cc-csc"
                  placeholder="123"
                  data-testid="card-cvc"
                  :class="{ 'input-invalid': cvcError }"
                  @input="onCvcInput"
                  @blur="touched.cvc = true"
                />
                <p v-if="cvcError" class="field-error" data-testid="card-cvc-error">{{ cvcError }}</p>
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
              <p v-if="holderError" class="field-error" data-testid="card-holder-error">
                {{ holderError }}
              </p>
            </div>

            <div class="summary-row" data-testid="checkout-row-subtotal">
              <span>Сабтотал</span>
              <span data-testid="checkout-subtotal">{{ formatMoney(merchandiseSubtotal) }}</span>
            </div>
            <div class="summary-row" data-testid="checkout-row-discount">
              <span>Скидка</span>
              <span data-testid="checkout-discount">{{ formatMoney(discountCents) }}</span>
            </div>
            <div class="summary-row" data-testid="checkout-row-delivery">
              <span>Доставка</span>
              <span data-testid="checkout-delivery-fee" :data-cents="deliveryFeeCents">
                {{ formatMoney(deliveryFeeCents) }}
              </span>
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
