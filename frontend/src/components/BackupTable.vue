<script setup lang="ts">
import BackupEditDialog from "@/components/BackupEditDialog.vue";
import BackupViewDialog from "@/components/BackupViewDialog.vue";
import { Backup, BackupStatus, DefaultService } from "@/models/api";
import { BackupType } from "@/models/api/models/BackupType";
import { useNotificationsStore } from "@/stores";
import { ref } from "vue";

const notificationsStore = useNotificationsStore();
const selectedItems = defineModel<Backup[]>();
const searchModel = defineModel<string | undefined>("search", { required: true });

const viewDialogID = ref<string | undefined>(undefined);
const editDialogID = ref<string | undefined>(undefined);

const filterDataTableItems = (value: string, query: string, item: any) => {
  if (query == null) return -1;
  if (!query.length) return 0;

  if (!value) {
    // when value is empty it is a surrogate column we assume it is the source column
    if (item.value.type === BackupType.BIG_QUERY) {
      value = item.value.bigquery_options?.dataset;
      if (item.value.bigquery_options?.table && item.value.bigquery_options?.table.length > 0) {
        value += item.value.bigquery_options?.table.join();
      }
    } else if (item.value.type === BackupType.CLOUD_STORAGE) {
      value = item.value.gcs_options?.bucket;
    }
  }

  value = value.toString().toLocaleLowerCase();
  query = query.toString().toLocaleLowerCase();

  const result = [];
  let idx = value.indexOf(query);
  while (~idx) {
    result.push([idx, idx + query.length] as const);

    idx = value.indexOf(query, idx + query.length);
  }

  return result.length ? result : -1;
};

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
  { title: "Data availability class", key: "data_availability_class" },
  { title: "Recovery requirements", key: "recovery_point_objective" },
  { title: "Created", key: "created" },
  { title: "Strategy", key: "strategy" },
  { title: "Status", key: "status" },
];

const updateData = async () => {
  isLoading.value = true;
  DefaultService.getBackups()
    .then((response) => {
      items.value = response.backups ?? [];
    })
    .catch((err) => {
      isLoading.value = false;
      notificationsStore.handleError(err);
    })
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
    v-model="selectedItems"
    v-model:search="searchModel"
    :items="items"
    :headers="headers"
    :custom-filter="filterDataTableItems"
    show-select
    :loading="isLoading"
    :items-per-page="25"
    show-current-page
    return-object
  >
    <template #header.data-table-select></template>
    <template #[`item.data-table-select`]="{ toggleSelect, item, internalItem, isSelected }">
      <v-checkbox
        v-if="[BackupStatus.PAUSED, BackupStatus.RUNNING, BackupStatus.NOT_STARTED, BackupStatus.FINISHED].includes(item?.status ?? '')"
        :model-value="isSelected(internalItem)"
        @click.stop="toggleSelect(internalItem)"
        color="primary"
        hide-details
      ></v-checkbox>
    </template>
    <template #[`item.edit`]="{ item }">
      <v-tooltip text="Edit Backup">
        <template #activator="{ props }">
          <v-icon
            v-bind="props"
            :disabled="[BackupStatus.BACKUP_DELETED, BackupStatus.BACKUP_SOURCE_DELETED].includes(item?.status ?? '')"
            @click="editDialogID = item.id"
            >mdi-wrench</v-icon
          >
        </template>
      </v-tooltip>
    </template>
    <template #[`item.view`]="{ item }">
      <v-tooltip text="View details">
        <template #activator="{ props }">
          <v-icon
            v-bind="props"
            :disabled="[BackupStatus.BACKUP_DELETED, BackupStatus.BACKUP_SOURCE_DELETED].includes(item?.status ?? '')"
            @click="viewDialogID = item.id"
            >mdi-view-list</v-icon
          >
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
    <template #[`item.created`]="{ item }">
      {{
        item.created
          ? new Date(item.created).toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" }) +
            ", " +
            new Date(item.created).toLocaleTimeString("en-US", {
              hour: "numeric",
              minute: "2-digit",
              second: "2-digit",
              hour12: true,
            })
          : ""
      }}
    </template>
    <template #[`item.data_availability_class`]="{ item }">
      {{ item.data_availability_class }}
    </template>
    <template #[`item.recovery_point_objective`]="{ item }">
      <ul>
        <li>RPO: {{ item.recovery_point_objective }} h</li>
        <li>RTO: {{ item.recovery_time_objective }} min</li>
      </ul>
    </template>
    <template #[`item.source`]="{ item }">
      <template v-if="item.type === BackupType.BIG_QUERY">
        BigQuery:
        <a :href="bigqueryDatasetLink(item.project ?? '', item.bigquery_options?.dataset ?? '')" target="_blank">{{
          item.bigquery_options?.dataset
        }}</a>
        <ul>
          <li v-for="(table, idx) in item.bigquery_options?.table" :key="idx">
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
