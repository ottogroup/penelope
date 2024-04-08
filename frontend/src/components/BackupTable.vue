<script setup lang="ts">
import BackupEditDialog from "@/components/BackupEditDialog.vue";
import BackupViewDialog from "@/components/BackupViewDialog.vue";
import { Backup, DefaultService } from "@/models/api";
import { BackupType } from "@/models/api/models/BackupType";
import { useNotificationsStore } from "@/stores";
import { ref } from "vue";

const notificationsStore = useNotificationsStore();

const selectedItems = defineModel();

const viewDialogID = ref<string | undefined>(undefined);
const editDialogID = ref<string | undefined>(undefined);

const isLoading = ref(true);
const items = ref<Backup[]>([]);
const headers = [
  { title: "", key: "edit", sortable: false },
  { title: "", key: "view", sortable: false },
  { title: "Type", key: "type" },
  { title: "Project", key: "project" },
  { title: "Strategy", key: "strategy" },
  { title: "Status", key: "status" },
  { title: "Sink bucket", key: "sink" },
  { title: "Source", key: "source" },
];

const updateData = async () => {
  isLoading.value = true;
  DefaultService.getBackups()
    .then((response) => {
      items.value = response.backups ?? [];
    })
    .catch((err) => notificationsStore.handleError(err))
    .finally(() => {
      isLoading.value = false;
    });
};

updateData();

const cloudStorageLink = (project: string, bucket: string) => {
  return `https://console.cloud.google.com/storage/browser/${bucket}?project=${project}`;
};

const bigqueryDatasetLink = (project: string, dataset: string) => {
  return `https://console.cloud.google.com/bigquery?project=${project}&p=${project}&d=${dataset}&page=dataset`;
};

const bigqueryTableLink = (project: string, dataset: string, table: string) => {
  return `https://console.cloud.google.com/bigquery?project=${project}&p=${project}&d=${dataset}&t=${table}&page=table`;
};
</script>

<template>
  <BackupViewDialog :id="viewDialogID" @close="viewDialogID = undefined" />
  <BackupEditDialog :id="editDialogID" @close="editDialogID = undefined" />

  <v-data-table
    :items="items"
    :headers="headers"
    show-select
    :loading="isLoading"
    v-model="selectedItems"
    :items-per-page="25"
  >
    <template #[`item.edit`]="{ item }">
      <v-icon @click="editDialogID = item.id">mdi-wrench</v-icon>
    </template>
    <template #[`item.view`]="{ item }">
      <v-icon @click="viewDialogID = item.id">mdi-view-list</v-icon>
    </template>
    <template #[`item.sink`]="{ item }">
      <a :href="cloudStorageLink(item.sink_project ?? '', item.sink ?? '')">{{ item.sink }}</a>
    </template>
    <template #[`item.source`]="{ item }">
      <template v-if="item.type === BackupType.BIG_QUERY">
        BigQuery:
        <a :href="bigqueryDatasetLink(item.project ?? '', item.bigquery_options?.dataset ?? '')">{{
          item.bigquery_options?.dataset
        }}</a>
        <ul>
          <li v-for="table in item.bigquery_options?.table">
            Table:
            <a :href="bigqueryTableLink(item.project ?? '', item.bigquery_options?.dataset ?? '', table)">{{
              table
            }}</a>
          </li>
        </ul>
      </template>
      <template v-if="item.type === BackupType.CLOUD_STORAGE">
        Bucket:
        <a :href="cloudStorageLink(item.project ?? '', item.gcs_options?.bucket ?? '')">{{
          item.gcs_options?.bucket
        }}</a>
      </template>
    </template>
  </v-data-table>
</template>
