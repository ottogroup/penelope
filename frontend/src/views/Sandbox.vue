<script setup lang="ts">
import Notifications from "@/components/common/Notifications.vue";
</script>

<template>
  <v-container :fluid="true" class="pa-0">
    <slot />
    <Notifications />
  </v-container>
</template>
