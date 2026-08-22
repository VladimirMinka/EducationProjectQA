export const PRODUCT_IMAGES = {
  "0001": "/images/products/0001.jpeg",
  "0002": "/images/products/0002.jpeg",
  "0003": "/images/products/0003.webp",
  "0004": "/images/products/0004.jpg",
  "0005": "/images/products/0005.jpg",
  "0006": "/images/products/0006.jpeg",
  "0007": "/images/products/0007.webp",
  "0009": "/images/products/0009.jpeg",
  "0012": "/images/products/0012.png",
  "0015": "/images/products/0015.jpeg",
};

export function productIdSuffix(id) {
  if (!id || typeof id !== "string") return "";
  return id.replace(/-/g, "").slice(-4).toLowerCase();
}

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
