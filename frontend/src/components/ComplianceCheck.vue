<script setup lang="ts">
import {BackupStatus, CreateRequest, DefaultService} from "@/models/api";
import {useNotificationsStore} from "@/stores";
import {computed, ref, watch} from "vue";
import Notification from "@/models/notification";

const notificationsStore = useNotificationsStore();

const props = defineProps<{ backup: CreateRequest | undefined }>();

const isLoading = ref(false);
const complianceChecks = ref<
  {
    field?: string;
    passed?: boolean;
    description?: string;
    details?: string;
  }[]
>([]);

const backupStatusIsDeleted = computed(() => {
  return props.backup?.status === BackupStatus.BACKUP_DELETED;
});

const updateData = () => {
  if (props.backup === undefined) {
    isLoading.value = false;
    complianceChecks.value = [];
    return;
  }
  isLoading.value = true;
  complianceChecks.value = [];

  if (!backupStatusIsDeleted.value) {
    DefaultService.postBackupsCompliance(props.backup)
      .then((resp) => {
        complianceChecks.value = resp.checks ?? [];
      })
      .catch(() => {
        notificationsStore.addNotification(
          new Notification({
            message: `Could not make compliance check for backup`,
            color: "warning",
          })
        );
      })
      .finally(() => {
        isLoading.value = false;
      });
  }
};

updateData();
watch(
  () => props.backup,
  () => {
    updateData();
  },
);
</script>

<template>
  <template v-if="!backupStatusIsDeleted && (complianceChecks.length > 0 || isLoading)">
    <h4>Compliance checks</h4>
    <v-progress-linear v-if="isLoading" indeterminate/>
    <v-list>
      <v-list-item v-for="check in complianceChecks">
        <v-list-item-title class="text-wrap">
          {{ check.description }}
        </v-list-item-title>
        <v-list-item-subtitle class="text-wrap">
          {{ check.details }}
        </v-list-item-subtitle>
        <template v-slot:append>
          <v-icon :color="check.passed ? 'success' : 'error'">
            {{ check.passed ? "mdi-check" : "mdi-close" }}
          </v-icon>
        </template>
      </v-list-item>
    </v-list>
  </template>
</template>
