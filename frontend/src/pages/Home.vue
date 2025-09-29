<script setup lang="ts">
import BackupCreateDialog from "@/components/BackupCreateDialog.vue";
import BackupTable from "@/components/BackupTable.vue";
import { Backup, BackupStatus, DefaultService } from "@/models/api";
import Notification from "@/models/notification";
import { useNotificationsStore } from "@/stores";
import { ref } from "vue";

const notificationsStore = useNotificationsStore();

const tableKey = ref(0);
const selectedItems = ref<Backup[]>([]);
const showCreateDialog = ref(false);
const search = ref("");

const onAddBackup = () => {
  showCreateDialog.value = true;
};

const onRefreshTable = () => {
  tableKey.value += 1;
};

const onPlay = () => {
  for (const backup of selectedItems.value) {
    notificationsStore.addNotification(
      new Notification({
        message: `Resuming backup ${backup.id}`,
        color: "info",
      }),
    );
    DefaultService.patchBackups({
      backup_id: backup.id,
      status: BackupStatus.NOT_STARTED,
    })
      .then(() => {
        notificationsStore.addNotification(
          new Notification({
            message: `Backup ${backup.id} resumed`,
            color: "success",
          }),
        );
      })
      .catch((err) => {
        notificationsStore.handleError(err);
      });
  }
};

const onPause = () => {
  for (const backup of selectedItems.value) {
    notificationsStore.addNotification(
      new Notification({
        message: `Pausing backup ${backup.id}`,
        color: "info",
      }),
    );
    DefaultService.patchBackups({
      backup_id: backup.id,
      status: BackupStatus.PAUSED,
    })
      .then(() => {
        notificationsStore.addNotification(
          new Notification({
            message: `Backup ${backup.id} paused`,
            color: "success",
          }),
        );
      })
      .catch((err) => {
        notificationsStore.handleError(err);
      });
  }
};

const onDelete = () => {
  for (const backup of selectedItems.value) {
    notificationsStore.addNotification(
      new Notification({
        message: `Deleting backup ${backup.id}`,
        color: "info",
      }),
    );
    DefaultService.patchBackups({
      backup_id: backup.id,
      status: BackupStatus.TO_DELETE,
    })
      .then(() => {
        notificationsStore.addNotification(
          new Notification({
            message: `Backup ${backup.id} deleted`,
            color: "success",
          }),
        );
      })
      .catch((err) => {
        notificationsStore.handleError(err);
      });
  }
};
</script>

<template>
  <BackupCreateDialog v-model="showCreateDialog" />

  <v-row align="center" class="ma-2">
    <v-col cols="auto">
      <v-btn-group>
        <v-btn @click="onAddBackup">
          <v-icon>mdi-plus</v-icon>
          Create Backup
        </v-btn>
        <v-btn @click="onRefreshTable">
          <v-icon>mdi-refresh</v-icon>
          Refresh
        </v-btn>
        <v-btn
          @click="onPlay"
          :disabled="selectedItems.length == 0 || selectedItems.some((item) => item.status !== BackupStatus.PAUSED)"
        >
          <v-icon>mdi-play</v-icon>
          Resume
        </v-btn>
        <v-btn
          @click="onPause"
          :disabled="
            selectedItems.length == 0 || selectedItems.some((item) => item.status !== BackupStatus.RUNNING)
          "
        >
          <v-icon>mdi-pause</v-icon>
          Pause
        </v-btn>
        <v-btn
          @click="onDelete"
          :disabled="selectedItems.length == 0 || selectedItems.some((item) => item.status !== BackupStatus.PAUSED)"
        >
          <v-icon>mdi-delete</v-icon>
          Delete
        </v-btn>
      </v-btn-group>
    </v-col>
    <v-spacer></v-spacer>
    <v-col cols="4">
      <v-text-field
        v-model="search"
        density="compact"
        label="Search"
        prepend-inner-icon="mdi-magnify"
        clearable
        flat
        hide-details
        single-line
      ></v-text-field>
    </v-col>
  </v-row>
  <BackupTable :key="tableKey" v-model="selectedItems" v-model:search="search" />
</template>
