import { createRouter, createWebHashHistory } from "vue-router";
import { getSession } from "../lib/auth";
import LoginView from "../views/LoginView.vue";
import CatalogView from "../views/CatalogView.vue";
import CartView from "../views/CartView.vue";
import CheckoutView from "../views/CheckoutView.vue";
import OrdersView from "../views/OrdersView.vue";

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: "/", redirect: "/catalog" },
    { path: "/login", name: "login", component: LoginView, meta: { public: true } },
    { path: "/catalog", name: "catalog", component: CatalogView },
    { path: "/cart", name: "cart", component: CartView },
    { path: "/checkout", name: "checkout", component: CheckoutView },
    { path: "/orders", name: "orders", component: OrdersView },
    { path: "/:pathMatch(.*)*", redirect: "/catalog" },
  ],
});

router.beforeEach((to) => {
  if (to.meta.public) {
    if (to.name === "login" && getSession()) return { name: "catalog" };
    return true;
  }
  if (!getSession()) return { name: "login" };
  return true;
});

export default router;
