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

let productsCache = null;
let productsInflight = null;

function listProductsCached() {
  if (productsCache) return Promise.resolve(productsCache);
  if (!productsInflight) {
    productsInflight = request("/v1/products")
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

export const api = {
  register: (payload) =>
    request("/v1/users/register", { method: "POST", body: payload }),
  login: (payload) =>
    request("/v1/users/login", { method: "POST", body: payload }),
  listProducts: () => listProductsCached(),
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
  createOrder: (userId, token) =>
    request("/v1/orders", {
      method: "POST",
      token,
      body: { user_id: userId },
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
