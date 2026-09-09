const API_BASE = window.__API_BASE__ || "";

export class ApiError extends Error {
  constructor(message, status, body) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

async function request(path, { method = "GET", body, token } = {}) {
  const headers = { Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  const text = await res.text();
  let data = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = { message: text };
    }
  }

  if (!res.ok) {
    const message =
      data?.message || data?.error || `Request failed (${res.status})`;
    throw new ApiError(message, res.status, data);
  }

  return data;
}

function cents(value) {
  const n = Number(value ?? 0);
  return Number.isFinite(n) ? n : 0;
}

export function formatMoney(centsValue) {
  return new Intl.NumberFormat("ru-RU", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
  }).format(cents(centsValue) / 100);
}

export const COURIER_FEE_CENTS = 29900;
export const FREE_DELIVERY_SUBTOTAL_CENTS = 500000;

export function deliveryFeeFor(method, subtotalCents) {
  if (method === "PICKUP" || method === "DELIVERY_METHOD_PICKUP") return 0;
  if (method === "COURIER" || method === "DELIVERY_METHOD_COURIER") {
    return cents(subtotalCents) >= FREE_DELIVERY_SUBTOTAL_CENTS ? 0 : COURIER_FEE_CENTS;
  }
  return 0;
}

function buildQuery(params = {}) {
  const q = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") return;
    q.set(key, String(value));
  });
  const s = q.toString();
  return s ? `?${s}` : "";
}

let productsCache = null;
let productsInflight = null;

function listProductsCached() {
  if (productsCache) return Promise.resolve(productsCache);
  if (!productsInflight) {
    productsInflight = request("/v1/products?page_size=50")
      .then((data) => {
        productsCache = data;
        productsInflight = null;
        return data;
      })
      .catch((err) => {
        productsInflight = null;
        throw err;
      });
  }
  return productsInflight;
}

export function invalidateProductsCache() {
  productsCache = null;
  productsInflight = null;
}

export const api = {
  register: (payload) =>
    request("/v1/users/register", { method: "POST", body: payload }),
  login: (payload) =>
    request("/v1/users/login", { method: "POST", body: payload }),
  listProducts: (params) => {
    if (!params || Object.keys(params).length === 0) {
      return listProductsCached();
    }
    return request(`/v1/products${buildQuery(params)}`);
  },
  listCategories: () => request("/v1/categories"),
  getCart: (userId, token) =>
    request(`/v1/users/${userId}/cart`, { token }),
  addItem: (userId, token, productId, quantity = 1) =>
    request(`/v1/users/${userId}/cart/items`, {
      method: "POST",
      token,
      body: { product_id: productId, quantity },
    }),
  removeItem: (userId, token, productId) =>
    request(`/v1/users/${userId}/cart/items/${productId}`, {
      method: "DELETE",
      token,
    }),
  clearCart: (userId, token) =>
    request(`/v1/users/${userId}/cart`, { method: "DELETE", token }),
  applyPromocode: (userId, token, code) =>
    request(`/v1/users/${userId}/cart/promocode`, {
      method: "POST",
      token,
      body: { code },
    }),
  clearPromocode: (userId, token) =>
    request(`/v1/users/${userId}/cart/promocode`, {
      method: "DELETE",
      token,
    }),
  listAddresses: (userId, token) =>
    request(`/v1/users/${userId}/addresses`, { token }),
  createAddress: (userId, token, payload) =>
    request(`/v1/users/${userId}/addresses`, {
      method: "POST",
      token,
      body: { user_id: userId, ...payload },
    }),
  deleteAddress: (userId, token, addressId) =>
    request(`/v1/users/${userId}/addresses/${addressId}`, {
      method: "DELETE",
      token,
    }),
  listPickupPoints: () => request("/v1/pickup-points"),
  createOrder: (userId, token, delivery) =>
    request("/v1/orders", {
      method: "POST",
      token,
      body: {
        user_id: userId,
        delivery_method: delivery.deliveryMethod,
        address_id: delivery.addressId || undefined,
        pickup_point_id: delivery.pickupPointId || undefined,
      },
    }),
  listOrders: (userId, token) =>
    request(`/v1/users/${userId}/orders`, { token }),
  getOrder: (orderId, token) => request(`/v1/orders/${orderId}`, { token }),
  cancelOrder: (orderId, token) =>
    request(`/v1/orders/${orderId}/cancel`, {
      method: "POST",
      token,
      body: {},
    }),
  updateOrderStatus: (orderId, token, fromStatus, toStatus) =>
    request(`/v1/orders/${orderId}/status`, {
      method: "POST",
      token,
      body: { fromStatus, toStatus },
    }),
};
