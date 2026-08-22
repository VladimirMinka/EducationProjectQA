<script setup>
import { computed, onMounted, ref, watch } from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";
import { getTheme, toggleTheme as flipTheme } from "../lib/theme";
import { useSession } from "../composables/useSession";
import { api } from "../lib/api";

const props = defineProps({
  active: { type: String, required: true },
});

const route = useRoute();
const router = useRouter();
const { session, clearSession } = useSession();
const theme = ref(getTheme());
const cartCount = ref(0);

const userLabel = computed(
  () => session.value?.user?.name || session.value?.user?.email || ""
);

async function refreshCartCount() {
  const s = session.value;
  if (!s) {
    cartCount.value = 0;
    return;
  }
  try {
    const cart = await api.getCart(s.user.id, s.accessToken);
    cartCount.value = (cart.items || []).reduce(
      (n, i) => n + Number(i.quantity || 0),
      0
    );
  } catch {
    cartCount.value = 0;
  }
}

function onThemeToggle() {
  theme.value = flipTheme();
}

function onLogout() {
  clearSession();
  router.push({ name: "login" });
}

onMounted(refreshCartCount);
watch(() => route.fullPath, refreshCartCount);

defineExpose({ refreshCartCount, setCartCount: (n) => { cartCount.value = Number(n) || 0; } });
</script>

<template>
  <div class="shell" data-testid="app-shell" :data-active-page="active">
    <header class="topbar" data-testid="topbar">
      <RouterLink class="brand" :to="{ name: 'catalog' }" data-testid="brand-link">
        <span class="brand-mark" aria-hidden="true" />
        <span data-testid="brand-name">STORE</span>
      </RouterLink>
      <nav class="nav" data-testid="main-nav">
        <RouterLink
          :to="{ name: 'catalog' }"
          data-testid="nav-catalog"
          data-nav="catalog"
          :class="{ active: active === 'catalog' }"
          :aria-current="active === 'catalog' ? 'page' : 'false'"
        >
          Каталог
        </RouterLink>
        <RouterLink
          :to="{ name: 'cart' }"
          class="cart-link"
          data-testid="nav-cart"
          data-nav="cart"
          :class="{ active: active === 'cart' }"
          :aria-current="active === 'cart' ? 'page' : 'false'"
        >
          Корзина
          <span
            class="badge"
            data-testid="cart-badge"
            :data-count="cartCount"
            :hidden="cartCount <= 0"
          >{{ cartCount }}</span>
        </RouterLink>
        <button
          type="button"
          class="linkish theme-toggle"
          data-testid="theme-toggle"
          :data-theme-current="theme"
          title="Тема"
          @click="onThemeToggle"
        >
          {{ theme === "dark" ? "Светлая" : "Тёмная" }}
        </button>
        <span class="user-label" data-testid="user-name">{{ userLabel }}</span>
        <button type="button" class="linkish" data-testid="logout-button" @click="onLogout">
          Выйти
        </button>
      </nav>
    </header>
    <div data-page-content data-testid="page-content">
      <slot />
    </div>
  </div>
</template>
