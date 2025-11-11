<script setup lang="ts">
import {Backup, BackupStatus, BackupStrategy, DefaultService, UpdateRequest} from "@/models/api";
import {BackupType} from "@/models/api/models/BackupType";
import Notification from "@/models/notification";
import {useNotificationsStore} from "@/stores";
import {computed, ref, watch} from "vue";

const notificationsStore = useNotificationsStore();

const props = defineProps({
  id: {
    type: String,
  },
});

const emits = defineEmits(['close']);
const viewDialog = ref(false);
const isLoading = ref(true);
const backup = ref<Backup>({
  gcs_options: {
    include_prefixes: [],
    exclude_prefixes: [],
  },
  bigquery_options: {
    table: [],
    excluded_tables: [],
  },
  target: {
    archive_ttm: 0,
  },
  mirror_options: {},
  snapshot_options: {},
  recovery_point_objective: 0,
  recovery_time_objective: 0,
});

const isValid = ref(false);

const backupStatusIsDeleted = computed(() => {
  return backup.value.status === BackupStatus.BACKUP_DELETED;
});

const updateData = () => {
  isLoading.value = true;
  if (props.id) {
    DefaultService.getSingleBackup(props.id!)
      .then((response) => {
        backup.value = response;
      })
      .catch((err) => {
        useNotificationsStore().handleError(err);
      })
      .finally(() => {
        isLoading.value = false;
      });
  }
};

const saveBackup = () => {
  if (!isValid.value) {
    return;
  }
  isLoading.value = true;
  const req: UpdateRequest = {
    backup_id: props.id!,
    mirror_ttl: Number(backup.value.mirror_options?.lifetime_in_days),
    snapshot_ttl: Number(backup.value.snapshot_options?.lifetime_in_days),
    archive_ttm: Number(backup.value.target?.archive_ttm),
    include_path: backup.value.gcs_options?.include_prefixes,
    exclude_path: backup.value.gcs_options?.exclude_prefixes,
    table: backup.value.bigquery_options?.table,
    excluded_tables: backup.value.bigquery_options?.excluded_tables,
    recovery_point_objective: Number(backup.value.recovery_point_objective),
    recovery_time_objective: Number(backup.value.recovery_time_objective),
  };

  DefaultService.patchBackups(req)
    .then(() => {
      notificationsStore.addNotification(
        new Notification({
          message: "Backup updated",
          color: "success",
        }),
      );
      viewDialog.value = false;
    })
    .catch((err) => notificationsStore.handleError(err))
    .finally(() => {
      isLoading.value = false;
    });
};

const integerRequiredRule = (fieldName: string) => {
  return (v: number) => (!!v && v > 0) || `${fieldName} is required and must be bigger than 0`;
};

watch(
  () => viewDialog.value,
  (value) => {
    if (!value) {
      emits('close');
    }
  },
);

watch(
  () => props.id,
  (id) => {
    viewDialog.value = !!id;
    updateData();
  },
);
</script>

<template>
  <v-dialog v-model="viewDialog" width="800">
    <v-card title="Update backup" :loading="isLoading">
      <v-card-text>
        <v-form :disabled="isLoading || backupStatusIsDeleted" v-model="isValid" fast-fail @submit.prevent>
          <v-row>
            <v-col>
              <h3>Source</h3>
              <v-text-field
                label="RPO (hours)*"
                type="number"
                hint="Recovery time objective: Minimal frequency a backup must be conducted."
                v-model="backup!.recovery_point_objective"
                :rules="[integerRequiredRule('Recovery point objective (hours)')]"
              ></v-text-field>
              <v-text-field
                label="RTO (minutes)*"
                type="number"
                hint="Recovery time objective: The recovery process time duration needed to restore data from backup storage."
                v-model="backup!.recovery_time_objective"
                :rules="[integerRequiredRule('Recovery time objective (minutes)')]"
              ></v-text-field>
              <template v-if="backup?.type == BackupType.CLOUD_STORAGE">
                <v-combobox
                  chips
                  multiple
                  clearable
                  label="Include paths"
                  v-model="backup!.gcs_options!.include_prefixes"
                ></v-combobox>
                <v-combobox
                  chips
                  multiple
                  clearable
                  label="Exclude paths"
                  v-model="backup!.gcs_options!.exclude_prefixes"
                ></v-combobox>
              </template>
              <template v-if="backup?.type == BackupType.BIG_QUERY">
                <v-combobox
                  chips
                  multiple
                  clearable
                  label="BigQuery tables"
                  hint="When empty will take all tables."
                  v-model="backup!.bigquery_options!.table"
                ></v-combobox>
                <v-combobox
                  chips
                  multiple
                  clearable
                  label="BigQuery excluded tables"
                  hint="When present will ignore given tables."
                  v-model="backup!.bigquery_options!.excluded_tables"
                ></v-combobox>
              </template>
            </v-col>
            <v-col>
              <h3>Target</h3>
              <v-text-field
                label="Archive TTM"
                type="number"
                hint="After X days change object storage class to archive. Default is 0."
                v-model="backup!.target!.archive_ttm"
              ></v-text-field>
            </v-col>
            <v-col>
              <h3>Details</h3>
              <template v-if="backup?.strategy == BackupStrategy.SNAPSHOT">
                <v-text-field
                  label="Snapshot TTL"
                  type="number"
                  hint="After X days data will be deleted. Default is 0."
                  v-model="backup!.snapshot_options!.lifetime_in_days"
                ></v-text-field>
              </template>
              <template v-if="backup?.strategy == BackupStrategy.ONESHOT">
                <v-text-field
                  label="Oneshot TTL"
                  type="number"
                  hint="After X days data will be deleted. Default is 0."
                  v-model="backup!.snapshot_options!.lifetime_in_days"
                ></v-text-field>
              </template>
              <template v-if="backup?.strategy == BackupStrategy.MIRROR">
                <v-text-field
                  label="Mirror TTL"
                  type="number"
                  hint="After X days data will be deleted. Default is 0."
                  v-model="backup!.mirror_options!.lifetime_in_days"
                ></v-text-field>
              </template>
            </v-col>
          </v-row>
        </v-form>
      </v-card-text>
      <template v-slot:actions>
        <v-btn-group class="ms-auto">
          <v-btn text="Cancel" @click="viewDialog = false"></v-btn>
          <v-btn text="Update" :disabled="isLoading || !isValid" @click="saveBackup()"></v-btn>
        </v-btn-group>
      </template>
    </v-card>
  </v-dialog>
</template>
