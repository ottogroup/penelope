<script setup lang="ts">
import BackupTable from "@/components/BackupTable.vue";
import { ref } from "vue";
import { useNotificationsStore } from '@/stores';
import Notification from "@/models/notification";
import { DefaultService } from "@/models/api";
import { onTestFailed } from "vitest";

const notificationsStore = useNotificationsStore();

const tableKey = ref(0);
const selectedItems = ref([]);

const onAddBackup = () => {
  console.log("Add Backup");
};

const onRefreshTable = () => {
  tableKey.value += 1;
};

const onPlay = () => {
  for(const backup of selectedItems.value) {
    notificationsStore.addNotification(
      new Notification({
        message: `Resuming backup ${backup}`,
        color: "info",
      }));
      DefaultService.patchBackups({
        backup_id: backup,
        status: "NotStarted",
      }).then(() => {
        notificationsStore.addNotification(
          new Notification({
            message: `Backup ${backup} resumed`,
            color: "success",
          }));
      }).catch((err) => {
        notificationsStore.handleError(err);
      });
  }
};

const onPause = () => {
  for(const backup of selectedItems.value) {
    notificationsStore.addNotification(
      new Notification({
        message: `Pausing backup ${backup}`,
        color: "info",
      }));
      DefaultService.patchBackups({
        backup_id: backup,
        status: "Paused",
      }).then(() => {
        notificationsStore.addNotification(
          new Notification({
            message: `Backup ${backup} paused`,
            color: "success",
          }));
      }).catch((err) => {
        notificationsStore.handleError(err);
      });
  }
};

const onDelete = () => {
  for(const backup of selectedItems.value) {
    notificationsStore.addNotification(
      new Notification({
        message: `Deleting backup ${backup}`,
        color: "info",
      }));
      DefaultService.patchBackups({
        backup_id: backup,
        status: "ToDelete",
      }).then(() => {
        notificationsStore.addNotification(
          new Notification({
            message: `Backup ${backup} deleted`,
            color: "success",
          }));
      }).catch((err) => {
        notificationsStore.handleError(err);
      });
  }
};
</script>

<template>
  <v-btn-group class="ma-2">
    <v-btn @click="onAddBackup">
      <v-icon>mdi-plus</v-icon>
      Create Backup
    </v-btn>
    <v-btn @click="onRefreshTable">
      <v-icon>mdi-refresh</v-icon>
      Refresh
    </v-btn>
    <v-btn @click="onPlay" :disabled="selectedItems.length == 0">
      <v-icon>mdi-play</v-icon>
      Resume
    </v-btn>
    <v-btn @click="onPause" :disabled="selectedItems.length == 0">
      <v-icon>mdi-pause</v-icon>
      Pause
    </v-btn>
    <v-btn @click="onDelete" :disabled="selectedItems.length == 0">
      <v-icon>mdi-delete</v-icon>
      Delete
    </v-btn>
  </v-btn-group>
  <BackupTable :key="tableKey" v-model="selectedItems" />
</template>