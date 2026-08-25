export function digitsOnly(value) {
  return String(value || "").replace(/\D/g, "");
}

/** ISO/IEC 7812 Luhn checksum. */
export function luhnCheck(pan) {
  const digits = digitsOnly(pan);
  if (digits.length < 13 || digits.length > 19) return false;
  let sum = 0;
  let doubleIt = false;
  for (let i = digits.length - 1; i >= 0; i -= 1) {
    let n = digits.charCodeAt(i) - 48;
    if (n < 0 || n > 9) return false;
    if (doubleIt) {
      n *= 2;
      if (n > 9) n -= 9;
    }
    sum += n;
    doubleIt = !doubleIt;
  }
  return sum % 10 === 0;
}

export function formatCardNumber(value) {
  const digits = digitsOnly(value).slice(0, 19);
  return groupCardDigits(digits);
}

function groupCardDigits(digits) {
  return String(digits || "")
    .replace(/(.{4})/g, "$1 ")
    .trim();
}

/** Show BIN (1–6) and last digits; mask 7–12. */
export function maskCardNumber(value) {
  const digits = digitsOnly(value);
  if (!digits.length) return "";
  const chars = digits.split("").map((d, i) => {
    const n = i + 1;
    return n >= 7 && n <= 12 ? "•" : d;
  });
  return groupCardDigits(chars.join(""));
}

export function mergeMaskedPanInput(displayed, storedValue) {
  const compact = String(displayed || "").replace(/\s/g, "");
  const stored = digitsOnly(storedValue);
  let out = "";
  let i = 0;
  for (const ch of compact) {
    if (ch === "•") {
      if (i < stored.length) out += stored[i];
      i += 1;
    } else if (ch >= "0" && ch <= "9") {
      out += ch;
      i += 1;
    }
  }
  return out.slice(0, 19);
}

export function secureProgram(brand) {
  switch (brand) {
    case "visa":
      return { name: "Visa Secure", host: "secure.visa.com" };
    case "mastercard":
      return { name: "Mastercard Identity Check", host: "secure.mastercard.com" };
    case "amex":
      return { name: "American Express SafeKey", host: "safekey.americanexpress.com" };
    case "mir":
      return { name: "Mir Accept", host: "secure.mironline.ru" };
    case "humo":
      return { name: "Humo 3-D Secure", host: "secure.humocard.uz" };
    case "unionpay":
      return { name: "UnionPay 3-D Secure", host: "secure.unionpay.com" };
    default:
      return { name: "3-D Secure", host: "secure-acs.cardinalcommerce.com" };
  }
}

export function formatExpiry(value) {
  const digits = digitsOnly(value).slice(0, 4);
  if (digits.length <= 2) return digits;
  return `${digits.slice(0, 2)}/${digits.slice(2)}`;
}

export function isExpiryValid(mmYy, now = new Date()) {
  const match = /^(\d{2})\s*\/\s*(\d{2})$/.exec(String(mmYy || "").trim());
  if (!match) return false;
  const month = Number(match[1]);
  const year = 2000 + Number(match[2]);
  if (month < 1 || month > 12) return false;
  const lastDay = new Date(year, month, 0, 23, 59, 59, 999);
  return lastDay >= now;
}

export function detectBrand(pan) {
  const digits = digitsOnly(pan);
  if (!digits) return "";
  if (digits.startsWith("9860")) return "humo";
  // MIR IIN 2200–2204; keep 220… as MIR while typing
  if (/^220[0-4]/.test(digits) || (digits.length < 4 && digits.startsWith("220"))) {
    return "mir";
  }
  if (/^5[1-5]/.test(digits)) return "mastercard";
  if (digits.length >= 4 && /^(222[1-9]|22[3-9]\d|2[3-6]\d{2}|27[01]\d|2720)/.test(digits)) {
    return "mastercard";
  }
  if (digits.startsWith("62")) return "unionpay";
  if (digits.startsWith("4")) return "visa";
  return "";
}

export function brandLabel(brand) {
  switch (brand) {
    case "visa":
      return "Visa";
    case "mastercard":
      return "Mastercard";
    case "mir":
      return "МИР";
    case "unionpay":
      return "UnionPay";
    case "humo":
      return "Humo";
    case "amex":
      return "Amex";
    default:
      return "";
  }
}

export const PAY_SCHEMES = [
  { id: "mastercard", label: "Mastercard", src: "/images/pay/mastercard.svg" },
  { id: "visa", label: "Visa", src: "/images/pay/visa.svg" },
  { id: "mir", label: "МИР", src: "/images/pay/mir.svg" },
  { id: "unionpay", label: "UnionPay", src: "/images/pay/unionpay.svg" },
  { id: "humo", label: "Humo", src: "/images/pay/humo.svg" },
];

export function cvcLengthForBrand(brand) {
  return brand === "amex" ? 4 : 3;
}
