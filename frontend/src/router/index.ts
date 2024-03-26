import { usePrincipalStore } from "@/stores";
import { RouteRecordRaw, createRouter, createWebHistory } from "vue-router";

const routes: RouteRecordRaw[] = [
  {
    path: "/",
    name: "home",
    component: () => import("@/pages/Home.vue"),
  },
  {
    path: "/login",
    name: "login",
    component: () => import("@/views/Login.vue"),
  },
  // otherwise, redirect to home
  {
    path: "/:pathMatch(.*)*",
    redirect: "/",
    meta: {
      drawer: {
        visible: false,
      },
    },
  },
];

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
});

const debug = import.meta.env.VITE_ENV !== "prod";
router.beforeResolve((to, from, next) => {
  const store = usePrincipalStore();
  const currentUser = store.principal;

  // route to login page when user is not authenticated
  if (to.name != "login" && (!currentUser || !currentUser.isValid())) {
    if (debug) {
      console.warn(
        `${from.path} -> ${to.path}: user not authenticated, redirect to /login with returnUrl: ${
          to.path
        } and query: ${Object.keys(to.query)
          .map((key) => `${key}=${to.query[key]?.toString() || ""}`)
          .join("&")}`,
      );
    }
    return next({ name: "login" });
  }
  next();
});

export default router;
