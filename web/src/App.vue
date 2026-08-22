<script setup>
import { RouterView } from "vue-router";
import { useToast } from "./composables/useToast";

const { toast, clearToast } = useToast();
</script>

<template>
  <RouterView />
  <div
    v-if="toast"
    class="toast"
    data-testid="toast"
    :data-toast-type="toast.type"
    :style="toast.type === 'error' ? { borderColor: 'rgba(255, 107, 122, 0.45)' } : undefined"
    @click="clearToast"
  >
    {{ toast.message }}
  </div>
</template>
