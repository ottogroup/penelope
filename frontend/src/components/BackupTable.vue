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
  { title: "Source", key: "source" },
  { title: "Sink Project", key: "sink_project" },
  { title: "Sink bucket", key: "sink" },
  { title: "Strategy", key: "strategy" },
  { title: "Status", key: "status" },
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

const projectLink = (project: string) => {
  return `https://console.cloud.google.com/welcome?project=${project}`;
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
      <v-tooltip text="Edit Backup">
        <template #activator="{ props }">
          <v-icon v-bind="props" @click="editDialogID = item.id">mdi-wrench</v-icon>
        </template>
      </v-tooltip>
    </template>
    <template #[`item.view`]="{ item }">
      <v-tooltip text="View details">
        <template #activator="{ props }">
          <v-icon v-bind="props" @click="viewDialogID = item.id">mdi-view-list</v-icon>
        </template>
      </v-tooltip>
    </template>
    <template #[`item.project`]="{ item }">
      <a :href="projectLink(item.project ?? '')" target="_blank">{{ item.project }}</a>
    </template>
    <template #[`item.sink`]="{ item }">
      <a :href="cloudStorageLink(item.sink_project ?? '', item.sink ?? '')" target="_blank">{{ item.sink }}</a>
    </template>
    <template #[`item.sink_project`]="{ item }">
      <a :href="projectLink(item.sink_project ?? '')" target="_blank">{{ item.sink_project }} </a>
    </template>
    <template #[`item.source`]="{ item }">
      <template v-if="item.type === BackupType.BIG_QUERY">
        BigQuery:
        <a :href="bigqueryDatasetLink(item.project ?? '', item.bigquery_options?.dataset ?? '')" target="_blank">{{
          item.bigquery_options?.dataset
        }}</a>
        <ul>
          <li v-for="table in item.bigquery_options?.table">
            Table:
            <a
              :href="bigqueryTableLink(item.project ?? '', item.bigquery_options?.dataset ?? '', table)"
              target="_blank"
              >{{ table }}</a
            >
          </li>
        </ul>
      </template>
      <template v-if="item.type === BackupType.CLOUD_STORAGE">
        Bucket:
        <a :href="cloudStorageLink(item.project ?? '', item.gcs_options?.bucket ?? '')" target="_blank">{{
          item.gcs_options?.bucket
        }}</a>
      </template>
    </template>
  </v-data-table>
</template>
