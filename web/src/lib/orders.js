const STATUS_BY_NUMBER = {
  1: "CREATED",
  2: "PAID",
  3: "SHIPPED",
  4: "CANCELLED",
  5: "COMPLETED",
};

const STATUS_LABELS = {
  CREATED: "Ожидает оплаты",
  PAID: "Оплачен",
  SHIPPED: "В доставке",
  CANCELLED: "Отменён",
  COMPLETED: "Доставлен",
};

export function orderStatusKey(status) {
  if (typeof status === "number") return STATUS_BY_NUMBER[status] || "UNKNOWN";
  const raw = String(status || "").toUpperCase();
  if (raw.includes("CREATED")) return "CREATED";
  if (raw.includes("CANCELLED") || raw.includes("CANCELED")) return "CANCELLED";
  if (raw.includes("COMPLETED")) return "COMPLETED";
  if (raw.includes("SHIPPED")) return "SHIPPED";
  if (raw.includes("PAID")) return "PAID";
  return "UNKNOWN";
}

export function orderStatusLabel(status) {
  const key = orderStatusKey(status);
  return STATUS_LABELS[key] || key;
}

export function deliveryMethodLabel(method) {
  const raw = String(method || "").toUpperCase();
  if (raw.includes("COURIER") || raw === "1") return "Курьер";
  if (raw.includes("PICKUP") || raw === "2") return "Самовывоз";
  return "—";
}

export function formatDeliveryInfo(order) {
  const info = order?.deliveryInfo || {};
  if (info.pickupCode) {
    return `${info.pickupCode}: ${info.pickupAddress || info.addressLine || ""}`;
  }
  return info.addressLine || [info.city, info.street, info.building].filter(Boolean).join(", ");
}

export function formatOrderDate(value) {
  if (!value) return "";
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? String(value) : d.toLocaleString("ru-RU");
}
