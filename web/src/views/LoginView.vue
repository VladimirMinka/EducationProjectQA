<script setup>
import { ref } from "vue";
import { useRouter } from "vue-router";
import { api } from "../lib/api";
import { getTheme, toggleTheme as flipTheme } from "../lib/theme";
import { useSession } from "../composables/useSession";

const router = useRouter();
const { setSession } = useSession();

const mode = ref("login");
const error = ref("");
const busy = ref(false);
const theme = ref(getTheme());
const form = ref({
  name: "",
  email: "",
  password: "",
});

function switchMode(next) {
  mode.value = next;
  error.value = "";
}

function onThemeToggle() {
  theme.value = flipTheme();
}

async function onSubmit() {
  if (busy.value) return;
  error.value = "";
  busy.value = true;
  try {
    const auth =
      mode.value === "login"
        ? await api.login({
            email: form.value.email.trim(),
            password: form.value.password,
          })
        : await api.register({
            email: form.value.email.trim(),
            password: form.value.password,
            name: form.value.name.trim(),
          });
    setSession(auth);
    router.push({ name: "catalog" });
  } catch (err) {
    error.value = err.message || "Не удалось войти";
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <main class="auth-page" data-testid="login-page" :data-auth-mode="mode">
    <button
      type="button"
      class="theme-fab linkish"
      data-testid="theme-toggle"
      :data-theme-current="theme"
      @click="onThemeToggle"
    >
      {{ theme === "dark" ? "Светлая тема" : "Тёмная тема" }}
    </button>
    <div class="auth-stage" data-testid="auth-stage">
      <h1 class="auth-brand" data-testid="auth-brand">STORE</h1>
      <p class="auth-lead" data-testid="auth-lead">
        Техника Apple, Samsung, NVIDIA и AMD — учебный магазин для API и UI.
      </p>
      <div class="panel" data-testid="auth-panel">
        <div class="tabs" role="tablist" data-testid="auth-tabs">
          <button
            type="button"
            class="tab"
            :class="{ active: mode === 'login' }"
            data-mode="login"
            data-testid="tab-login"
            role="tab"
            :aria-selected="mode === 'login'"
            @click="switchMode('login')"
          >
            Вход
          </button>
          <button
            type="button"
            class="tab"
            :class="{ active: mode === 'register' }"
            data-mode="register"
            data-testid="tab-register"
            role="tab"
            :aria-selected="mode === 'register'"
            @click="switchMode('register')"
          >
            Регистрация
          </button>
        </div>
        <div v-if="error" class="alert alert-error" data-testid="auth-error" role="alert">
          {{ error }}
        </div>
        <form data-testid="auth-form" :data-auth-mode="mode" @submit.prevent="onSubmit">
          <div v-if="mode === 'register'" class="field">
            <label for="name">Имя</label>
            <input
              id="name"
              v-model="form.name"
              name="name"
              autocomplete="name"
              required
              data-testid="register-name"
            />
          </div>
          <div class="field">
            <label for="email">Email</label>
            <input
              id="email"
              v-model="form.email"
              name="email"
              type="email"
              autocomplete="username"
              required
              data-testid="login-email"
            />
          </div>
          <div class="field">
            <label for="password">Пароль</label>
            <input
              id="password"
              v-model="form.password"
              name="password"
              type="password"
              :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
              required
              minlength="6"
              data-testid="login-password"
            />
          </div>
          <button
            class="btn btn-primary"
            type="submit"
            :disabled="busy"
            data-testid="auth-submit"
            :data-auth-action="mode"
          >
            {{ busy ? "…" : mode === "login" ? "Войти" : "Создать аккаунт" }}
          </button>
        </form>
        <p class="hint" data-testid="auth-hint">
          Сидовый админ:
          <code data-testid="seed-admin-email">admin@store.local</code> /
          <code data-testid="seed-admin-password">admin123</code>
        </p>
      </div>
    </div>
  </main>
</template>
