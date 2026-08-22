import { createApp } from "vue";
import App from "./App.vue";
import router from "./router";
import { initTheme } from "./lib/theme";
import "./assets/styles.css";

initTheme();

createApp(App).use(router).mount("#app");
