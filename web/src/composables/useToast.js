import { computed, reactive } from "vue";

const state = reactive({
  toast: null,
});

let timer = null;

export function useToast() {
  function showToast(message, isError = false) {
    clearTimeout(timer);
    state.toast = {
      message,
      type: isError ? "error" : "ok",
    };
    timer = setTimeout(() => {
      state.toast = null;
    }, 2800);
  }

  function clearToast() {
    clearTimeout(timer);
    state.toast = null;
  }

  return {
    toast: computed(() => state.toast),
    showToast,
    clearToast,
  };
}
