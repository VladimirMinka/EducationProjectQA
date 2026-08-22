<script setup>
import { computed, ref, watch } from "vue";
import { productImageCandidates, productIdSuffix } from "../lib/images";

const props = defineProps({
  product: { type: Object, required: true },
  alt: { type: String, default: "" },
  thumb: { type: Boolean, default: false },
});

const candidates = computed(() => productImageCandidates(props.product));
const index = ref(0);
const broken = ref(false);

watch(
  () => props.product?.id,
  () => {
    index.value = 0;
    broken.value = false;
  }
);

const src = computed(() => candidates.value[index.value] || "/images/brands/unknown.svg");
const hasPhoto = computed(() => !String(src.value).includes("/brands/"));
const suffix = computed(() => productIdSuffix(props.product?.id));

function onError() {
  if (index.value < candidates.value.length - 1) {
    index.value += 1;
    return;
  }
  broken.value = true;
}
</script>

<template>
  <div
    v-if="!thumb"
    class="product-media"
    :class="{ 'has-photo': hasPhoto }"
    data-testid="product-media"
  >
    <img
      class="product-image"
      :class="{ 'is-broken': broken }"
      :src="src"
      :alt="alt || product.name || 'product'"
      width="400"
      height="300"
      data-testid="product-image"
      loading="lazy"
      decoding="async"
      @error="onError"
    />
  </div>
  <img
    v-else
    class="cart-thumb"
    :class="{ 'is-broken': broken }"
    :src="src"
    :alt="alt || product.name || 'product'"
    data-testid="cart-item-image"
    loading="lazy"
    @error="onError"
  />
</template>
