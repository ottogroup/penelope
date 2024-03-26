<script setup lang="ts">
import { truncate } from "@/helpers/filters";
import { useNotificationsStore } from "@/stores";

defineProps({
  timeout: {
    type: [Number, String],
    default: 5000,
  },
});

const notificationsStore = useNotificationsStore();

const onChanged = (id: string, model: boolean) => !model && notificationsStore.removeNotification(id);
</script>

<template>
  <v-snackbar
    v-for="snackbar in notificationsStore.notifications"
    :key="snackbar.id"
    v-model="snackbar.model"
    variant="flat"
    location="top center"
    :multi-line="true"
    :color="snackbar.color"
    :timeout="timeout"
    :style="{ top: `${snackbar.position + 64}px` }"
    @update:model-value="onChanged(snackbar.id, $event)"
  >
    {{ truncate(snackbar.message, 1024, "...") }}
    <template #actions>
      <v-btn variant="text" color="primary" @click="notificationsStore.removeNotification(snackbar.id)"> close</v-btn>
    </template>
  </v-snackbar>
</template>
