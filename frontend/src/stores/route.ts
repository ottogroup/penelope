import { defineStore } from "pinia";
import { Ref, ref } from "vue";

export const useRouteStore = defineStore("route", () => {
  const returnUrl: Ref<string> = ref("");

  function finishRouting() {
  }

  return { returnUrl, finishRouting };
});
