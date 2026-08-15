/**
 * Optional per-product photos.
 * Key = last 4 hex digits of product UUID (0001…0050).
 * Drop files into /images/products/ and register them here.
 */
export const PRODUCT_IMAGES = {
  "0001": "/images/products/0001.jpeg", // iPhone 15
  "0002": "/images/products/0002.jpeg", // iPhone 15 Pro
  "0003": "/images/products/0003.webp", // iPhone 15 Pro Max
  "0004": "/images/products/0004.jpg", // MacBook Air 13 M3
  "0005": "/images/products/0005.jpg", // MacBook Pro 14 M3
  "0006": "/images/products/0006.jpeg", // MacBook Pro 16 M3 Max
  "0007": "/images/products/0007.webp", // iPad Pro 11
  "0009": "/images/products/0009.jpeg", // iPad mini
  "0012": "/images/products/0012.png", // Apple Watch Ultra 2
  "0015": "/images/products/0015.jpeg", // Magic Keyboard
};

/**
 * Fallback chain:
 *   1) registered product image (if any)
 *   2) /images/brands/{brand}.svg
 *   3) /images/brands/unknown.svg
 */
export function productImageCandidates(product) {
  const suffix = productIdSuffix(product?.id);
  const brand = String(product?.brand || "unknown").toLowerCase();
  const files = [];
  if (suffix && PRODUCT_IMAGES[suffix]) {
    files.push(PRODUCT_IMAGES[suffix]);
  }
  files.push(`/images/brands/${brand}.svg`);
  files.push(`/images/brands/unknown.svg`);
  return files;
}

export function productIdSuffix(id) {
  if (!id || typeof id !== "string") return "";
  const hex = id.replace(/-/g, "");
  return hex.slice(-4).toLowerCase();
}

export function productImageAttrs(product, { alt } = {}) {
  const candidates = productImageCandidates(product);
  const primary = candidates[0];
  const rest = JSON.stringify(candidates.slice(1));
  return {
    src: primary,
    alt: alt || product?.name || "product",
    "data-fallbacks": rest,
  };
}

export function bindImageFallbacks(root) {
  root.querySelectorAll("img[data-fallbacks]").forEach((img) => {
    let queue = [];
    try {
      queue = JSON.parse(img.dataset.fallbacks || "[]");
    } catch {
      queue = [];
    }
    img.addEventListener("error", function onErr() {
      if (!queue.length) {
        img.removeEventListener("error", onErr);
        img.classList.add("is-broken");
        return;
      }
      img.src = queue.shift();
      img.dataset.fallbacks = JSON.stringify(queue);
    });
  });
}
