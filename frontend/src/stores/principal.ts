import Principal from "@/models/principal";
import { defineStore } from "pinia";
import { reactive } from "vue";

export const usePrincipalStore = defineStore("principal", () => {
  const principal = reactive(new Principal({}));

  function finalizePrincipal() {}

  return { principal, finalizePrincipal };
});
