<script setup lang="ts">
import logo from "@/assets/penelope_32.png";
import Sandbox from "@/views/Sandbox.vue";
import { ref } from "vue";

import { usePrincipalStore } from "./stores";

const principalStore = usePrincipalStore();

const title = ref("");
if (import.meta.env.VITE_ENV !== "prod") {
  title.value = `${import.meta.env.VITE_APP_TITLE} (${import.meta.env.VITE_ENV})`;
} else {
  title.value = `${import.meta.env.VITE_APP_TITLE}`;
}
document.title = title.value;
</script>

<template>
  <v-app>
    <v-app-bar color="app-bar" app>
      <template #prepend>
        <router-link to="/" class="ml-2">
          <v-avatar color="white">
            <v-img :src="logo" />
          </v-avatar>
        </router-link>
        &nbsp;
        {{ title }}
      </template>

      <template #append>
        <v-tooltip text="Documentation" location="bottom">
          <template #activator="{ props }">
            <v-btn
              v-bind="props"
              href="https://github.com/ottogroup/penelope/wiki"
              target="_blank"
              icon="mdi-school-outline"
              class="mr-2"
            />
          </template>
        </v-tooltip>
        {{ principalStore.principal.getEmail() }}
      </template>
    </v-app-bar>

    <v-main class="bg-app-background">
      <Sandbox>
        <router-view />
      </Sandbox>
    </v-main>
  </v-app>
</template>

<style>
@import "../node_modules/@fontsource/roboto/index.css";

.v-toolbar__content {
  border-bottom: thin solid rgba(var(--v-theme-on-app-navigation), var(--v-border-opacity));
}
</style>
