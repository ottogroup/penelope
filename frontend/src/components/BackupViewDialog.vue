<script setup lang="ts">
import ComplianceCheck from "@/components/ComplianceCheck.vue";
import PricePrediction from "@/components/PricePrediction.vue";
import ConfirmDialog from "@/components/common/ConfirmDialog.vue";
import { Backup, CreateRequest, DefaultService, Job, JobStatus, TrashcanCleanupStatus } from "@/models/api";
import { BackupType } from "@/models/api/models/BackupType";
import { RestoreResponse } from "@/models/api/models/RestoreResponse";
import Notification from "@/models/notification";
import { useNotificationsStore } from "@/stores";
import { ref, watch } from "vue";

const props = defineProps({
  id: {
    type: String,
  },
});

const notificationsStore = useNotificationsStore();

const emits = defineEmits(["close"]);
const tab = ref();
const viewDialog = ref(false);
const isLoading = ref(true);
const listIsLoading = ref(true);
const backup = ref<Backup | undefined>(undefined);
const backupForEval = ref<CreateRequest | undefined>(undefined);
const jobItems = ref<{ job: Job; restore: RestoreResponse | undefined }[]>([]);
const cleanupTrashcanDialog = ref(false);

const confirmCleanupTrashcan = () => {
  cleanupTrashcanDialog.value = true;
};

const cleanupTrashcan = () => {
  if (backup.value?.id) {
    DefaultService.postTrashcansCleanUp(backup.value?.id)
      .then(() => {
        notificationsStore.addNotification(
          new Notification({
            message: "Backup trashcan cleaned up",
            color: "success",
          }),
        );
        cleanupTrashcanDialog.value = false;
        updateData();
      })
      .catch((err) => {
        notificationsStore.handleError(err);
      });
  }
};

const cancelCleanupTrashcan = () => {
  cleanupTrashcanDialog.value = false;
};

const updateData = () => {
  isLoading.value = true;
  if (props.id) {
    DefaultService.getBackups1(props.id!)
      .then((response) => {
        backup.value = response;
        backupForEval.value = {
          type: response.type,
          strategy: response.strategy,
          project: response.project,
          target: response.target,
          snapshot_options: response.snapshot_options,
          mirror_options: response.mirror_options,
          bigquery_options: response.bigquery_options,
          gcs_options: response.gcs_options,
          status: response.status,
        };
      })
      .catch((err) => {
        useNotificationsStore().handleError(err);
      })
      .finally(() => {
        isLoading.value = false;
      });
  }
};

const headers = [
  {
    title: "Status",
    key: "job.status",
  },
  {
    title: "Source",
    key: "job.source",
  },
  {
    title: "Updated",
    key: "job.updated",
  },
  {
    title: "Foreign Job ID",
    key: "job.foreign_job_id",
  },
  {
    title: "Actions",
    key: "action",
  },
];

const loadJobs = ({ page, itemsPerPage }: { page: number; itemsPerPage: number; sortBy: string }) => {
  listIsLoading.value = true;
  if (page === 1) {
    jobItems.value =
      backup.value?.jobs?.slice(0, itemsPerPage).map((j: Job) => {
        return { job: j, restore: undefined };
      }) ?? [];
    listIsLoading.value = false;
  } else {
    DefaultService.getBackups1(props.id!, itemsPerPage, page)
      .then((resp) => {
        jobItems.value =
          resp.jobs?.map((j: Job) => {
            return { job: j, restore: undefined };
          }) ?? [];
      })
      .catch((err) => {
        notificationsStore.handleError(err);
      })
      .finally(() => {
        listIsLoading.value = false;
      });
  }
};

const loadRestore = (item: { job: Job; restore: (RestoreResponse & { isLoading?: boolean }) | undefined }) => {
  item.restore = { isLoading: true };
  DefaultService.getRestore(backup.value?.id!, item.job.id)
    .then((resp) => {
      item.restore = resp;
      item.restore.isLoading = false;
    })
    .catch((err) => {
      item.restore = { isLoading: false };
      notificationsStore.handleError(err);
    })
    .finally(() => {
      if (item.restore) {
        item.restore.isLoading = false;
      } else {
        item.restore = { isLoading: false };
      }
    });
};

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

const translateTrashcanCleanupStatus = (status: TrashcanCleanupStatus | undefined) => {
  switch (status) {
    case TrashcanCleanupStatus.NOOP:
      return "No cleanup scheduled";
    case TrashcanCleanupStatus.ERROR:
      return "Error";
    case TrashcanCleanupStatus.SCHEDULED:
      return "Scheduled";
    default:
      return "Unknown";
  }
};

watch(
  () => viewDialog.value,
  (value) => {
    if (!value) {
      emits("close");
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
  <v-dialog v-model="viewDialog" width="800" v-if="!cleanupTrashcanDialog">
    <v-card title="Backup">
      <v-card-text v-if="isLoading">
        <v-progress-linear indeterminate />
      </v-card-text>
      <v-card-text v-else>
        <v-tabs v-model="tab">
          <v-tab value="details">Details</v-tab>
          <v-tab value="jobs">Jobs</v-tab>
        </v-tabs>
        <v-window v-model="tab">
          <v-window-item value="details">
            <v-table>
              <tbody>
                <tr>
                  <td colspan="2"><h4>Source</h4></td>
                </tr>
                <tr>
                  <td>Type:</td>
                  <td>{{ backup?.type }}</td>
                </tr>
                <tr>
                  <td>Project:</td>
                  <td>
                    <a :href="projectLink(backup?.project ?? '')" target="_blank">{{ backup?.project }}</a>
                  </td>
                </tr>
                <tr>
                  <td>Source:</td>
                  <td>
                    <template v-if="backup?.type === BackupType.BIG_QUERY">
                      BigQuery:
                      <a
                        :href="bigqueryDatasetLink(backup?.project ?? '', backup?.bigquery_options?.dataset ?? '')"
                        target="_blank"
                        >{{ backup?.bigquery_options?.dataset }}</a
                      >
                    </template>
                    <template v-if="backup?.type === BackupType.CLOUD_STORAGE">
                      Bucket:
                      <a
                        :href="cloudStorageLink(backup?.project ?? '', backup?.gcs_options?.bucket ?? '')"
                        target="_blank"
                        >{{ backup?.gcs_options?.bucket }}</a
                      >
                    </template>
                  </td>
                </tr>
                <tr v-if="backup?.type === BackupType.BIG_QUERY && (backup?.bigquery_options?.table?.length ?? 0 > 0)">
                  <td>Tables:</td>
                  <td>
                    <ul>
                      <li v-for="table in backup?.bigquery_options?.table">
                        Table:
                        <a
                          :href="
                            bigqueryTableLink(backup?.project ?? '', backup?.bigquery_options?.dataset ?? '', table)
                          "
                          target="_blank"
                          >{{ table }}</a
                        >
                      </li>
                    </ul>
                  </td>
                </tr>
                <tr
                  v-if="
                    backup?.type === BackupType.BIG_QUERY &&
                    (backup?.bigquery_options?.excluded_tables?.length ?? 0 > 0)
                  "
                >
                  <td>Excluded tables:</td>
                  <td>
                    <ul>
                      <li v-for="table in backup?.bigquery_options?.excluded_tables">
                        Table:
                        <a
                          :href="
                            bigqueryTableLink(backup?.project ?? '', backup?.bigquery_options?.dataset ?? '', table)
                          "
                          target="_blank"
                          >{{ table }}</a
                        >
                      </li>
                    </ul>
                  </td>
                </tr>
                <tr
                  v-if="
                    backup?.type === BackupType.CLOUD_STORAGE &&
                    (backup?.gcs_options?.include_prefixes?.length ?? 0 > 0)
                  "
                >
                  <td>Included prefixes:</td>
                  <td>
                    <ul>
                      <li v-for="prefix in backup?.gcs_options?.include_prefixes">
                        {{ prefix }}
                      </li>
                    </ul>
                  </td>
                </tr>
                <tr
                  v-if="
                    backup?.type === BackupType.CLOUD_STORAGE &&
                    (backup?.gcs_options?.exclude_prefixes?.length ?? 0 > 0)
                  "
                >
                  <td>Excluded prefixes:</td>
                  <td>
                    <ul>
                      <li v-for="prefix in backup?.gcs_options?.exclude_prefixes">
                        {{ prefix }}
                      </li>
                    </ul>
                  </td>
                </tr>
                <tr>
                  <td>Data owner:</td>
                  <td>{{ backup?.data_owner || "n/a" }}</td>
                </tr>
                <tr>
                  <td>Data availability class:</td>
                  <td>{{ backup?.data_availability_class || "n/a" }}</td>
                </tr>
                <tr>
                  <td>Recovery Point Objective (hours):</td>
                  <td>{{ backup?.recovery_point_objective || "n/a" }}</td>
                </tr>
                <tr>
                  <td>Recovery Time Objective (minutes):</td>
                  <td>{{ backup?.recovery_time_objective || "n/a" }}</td>
                </tr>
                <tr>
                  <td colspan="2"><h4>Target</h4></td>
                </tr>
                <tr>
                  <td>Sink Project:</td>
                  <td>
                    <a :href="projectLink(backup?.sink_project ?? '')" target="_blank">{{ backup?.sink_project }}</a>
                  </td>
                </tr>
                <tr>
                  <td>Sink bucket:</td>
                  <td>
                    <a :href="cloudStorageLink(backup?.sink_project ?? '', backup?.sink ?? '')" target="_blank">{{
                      backup?.sink
                    }}</a>
                  </td>
                </tr>
                <tr>
                  <td>Storage region:</td>
                  <td>{{ backup?.target?.region }}</td>
                </tr>
                <tr v-if="backup?.target?.dual_region">
                  <td>Secondary storage region:</td>
                  <td>{{ backup?.target?.dual_region }}</td>
                </tr>
                <tr>
                  <td>Storage class:</td>
                  <td>{{ backup?.target?.storage_class }}</td>
                </tr>
                <tr>
                  <td>Archive TTM:</td>
                  <td>{{ backup?.target?.archive_ttm }}</td>
                </tr>

                <tr>
                  <td colspan="2"><h4>Details</h4></td>
                </tr>
                <tr>
                  <td>Strategy:</td>
                  <td>{{ backup?.strategy }}</td>
                </tr>
                <tr v-if="backup?.snapshot_options?.frequency_in_hours !== undefined">
                  <td>Snapshot frequency:</td>
                  <td>{{ backup?.snapshot_options?.frequency_in_hours }}h</td>
                </tr>
                <tr v-if="backup?.snapshot_options?.lifetime_in_days">
                  <td>Lifetime:</td>
                  <td>{{ backup?.snapshot_options?.lifetime_in_days }} days</td>
                </tr>
                <tr v-if="backup?.snapshot_options?.last_scheduled">
                  <td>Snapshot last scheduled:</td>
                  <td>{{ backup?.snapshot_options?.last_scheduled }}</td>
                </tr>
                <tr v-if="backup?.mirror_options?.lifetime_in_days">
                  <td>Lifetime:</td>
                  <td>{{ backup?.snapshot_options?.lifetime_in_days }} days</td>
                </tr>
                <tr>
                  <td colspan="2"><h4>Status</h4></td>
                </tr>
                <tr>
                  <td>Status:</td>
                  <td>{{ backup?.status }}</td>
                </tr>
                <tr>
                  <td>Created:</td>
                  <td>{{ backup?.created }}</td>
                </tr>
                <tr>
                  <td>Updated:</td>
                  <td>{{ backup?.updated }}</td>
                </tr>
                <tr>
                  <td colspan="2"><h4>Trashcan Cleanup</h4></td>
                </tr>
                <tr>
                  <td>Status:</td>
                  <td>{{ translateTrashcanCleanupStatus(backup?.trashcan_cleanup_status) }}</td>
                </tr>
                <tr v-if="backup?.trashcan_cleanup_status === TrashcanCleanupStatus.ERROR">
                  <td>Error Message:</td>
                  <td>{{ backup?.trashcan_cleanup_error_message?.split(":").pop() }}</td>
                </tr>
                <tr v-if="backup?.trashcan_cleanup_last_scheduled_time">
                  <td>Scheduled:</td>
                  <td>{{ backup?.trashcan_cleanup_last_scheduled_time }}</td>
                </tr>
                <tr v-if="backup?.trashcan_cleanup_status !== TrashcanCleanupStatus.SCHEDULED">
                  <td>Executed:</td>
                  <td>
                    <v-btn v-bind="props" color="red" @click="confirmCleanupTrashcan()" variant="tonal">
                      <v-icon>mdi-delete</v-icon>
                      Trashcan
                    </v-btn>
                  </td>
                </tr>
              </tbody>
            </v-table>
            <v-row>
              <v-col>
                <PricePrediction :backup="backupForEval" />
              </v-col>
              <v-col>
                <ComplianceCheck :backup="backupForEval" />
              </v-col>
            </v-row>
          </v-window-item>
          <v-window-item value="jobs">
            <v-data-table-server
              @update:options="loadJobs"
              :items-length="backup?.jobs_total ?? 0"
              :items="jobItems"
              :headers="headers"
              :loading="listIsLoading"
              item-value="job.id"
            >
              <template #[`item.job.status`]="{ item }">
                    <v-icon color="warning" v-if="item.job.status === JobStatus.SCHEDULED"
                      >mdi-clock-outline</v-icon
                    >
                    <v-icon color="success" v-else-if="item.job.status === JobStatus.FINISHED_OK"
                      >mdi-check</v-icon
                    >
                    <v-icon
                      color="error"
                      v-else-if="
                        item.job.status === JobStatus.ERROR ||
                        item.job.status === JobStatus.FINISHED_ERROR ||
                        item.job.status === JobStatus.FINISHED_QUOTA_ERROR
                      "
                      >mdi-close-circle-outline
                    </v-icon>
                    <v-icon color="success" v-else-if="item.job.status === JobStatus.JOB_DELETED"
                      >mdi-check</v-icon
                    >
                    <v-icon color="grey"  v-else>mdi-close-circle-outline</v-icon>
              </template>
              <template #[`item.action`]="{ item, internalItem, toggleExpand, isExpanded }">
                <v-btn
                  v-if="backup?.type === BackupType.BIG_QUERY && !isExpanded(internalItem)"
                  variant="outlined"
                  @click="
                    loadRestore(item);
                    toggleExpand(internalItem);
                  "
                >
                  show restore commands
                </v-btn>
              </template>
              <template v-slot:expanded-row="{ columns, item }">
                <tr>
                  <td :colspan="columns.length">
                    <template
                      v-if="
                        item.restore &&
                        !item.restore?.isLoading &&
                        item.restore?.actions &&
                        item.restore?.actions.length > 0
                      "
                    >
                      <span v-for="action in item.restore?.actions">
                        <code> {{ action.action }}</code>
                        <br />
                      </span>
                    </template>
                    <template v-else-if="item.restore && item.restore.isLoading">
                      <v-progress-circular size="small" :indeterminate="true" color="primary" />
                      Loading restore commands
                    </template>
                    <template v-else> No restore commands available </template>
                  </td>
                </tr>
              </template>
            </v-data-table-server>
          </v-window-item>
        </v-window>
      </v-card-text>
      <template v-slot:actions>
        <v-btn class="ms-auto" text="Close" @click="viewDialog = false"></v-btn>
      </template>
    </v-card>
  </v-dialog>
  <ConfirmDialog
    v-model="cleanupTrashcanDialog"
    :options="{
      title: 'Cleanup trashcan',
      message: 'Are you sure you want to cleanup trashcan?',
      color: 'red',
      confirmButtonText: 'Cleanup',
      cancelButtonText: 'Cancel',
    }"
    @confirm="cleanupTrashcan"
    @cancel="cancelCleanupTrashcan"
  ></ConfirmDialog>
</template>
