<script setup lang="ts">
import ComplianceCheck from "@/components/ComplianceCheck.vue";
import PricePrediction from "@/components/PricePrediction.vue";
import ConfirmDialog from "@/components/common/ConfirmDialog.vue";
import { copyToClipboard } from "@/helpers/clipboard";
import { Backup, BackupStatus, CreateRequest, DefaultService, Job, JobStatus, TrashcanCleanupStatus } from "@/models/api";
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
const recoveryIsLoading = ref(false);
const backup = ref<Backup | undefined>(undefined);
const backupForEval = ref<CreateRequest | undefined>(undefined);
const jobItems = ref<{ job: Job; restore: RestoreResponse | undefined }[]>([]);
const recoverableJobItems = ref<{ job: Job; restore: RestoreResponse | undefined }[]>([]);
const restoreActions = ref<string[]>([]);
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
    DefaultService.getSingleBackup(props.id!)
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
    sortable: false,
  },
  {
    title: "Source",
    key: "job.source",
    sortable: false,
  },
  {
    title: "Updated",
    key: "job.updated",
    sortable: false,
  },
  {
    title: "Foreign Job ID",
    key: "job.foreign_job_id",
    sortable: false,
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
    DefaultService.getSingleBackup(props.id!, itemsPerPage, page-1)
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

const loadRecoverableJobs = ({ page, itemsPerPage }: { page: number; itemsPerPage: number; sortBy: string }) => {
  listIsLoading.value = true;
  if (page === 1) {
    recoverableJobItems.value =
      backup.value?.jobs?.slice(0, itemsPerPage).map((j: Job) => {
        return { job: j, restore: undefined };
      }) ?? [];
    listIsLoading.value = false;
  } else {
    DefaultService.getSingleBackup(props.id!, itemsPerPage, page-1, JobStatus.FINISHED_OK)
      .then((resp) => {
        recoverableJobItems.value =
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

const loadRestore = (jobId?: string) => {
  recoveryIsLoading.value = true;
  DefaultService.getRestore(backup.value?.id!, jobId)
    .then((resp) => {
      restoreActions.value = resp?.actions?.map((a) => a.action).filter(Boolean) ?? [];
      recoveryIsLoading.value = true;
    })
    .catch((err) => {
      restoreActions.value = [];
      recoveryIsLoading.value = false;
      notificationsStore.handleError(err);
    })
    .finally(() => {
      recoveryIsLoading.value = false;
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
      tab.value = "details";
      jobItems.value = [];
      restoreActions.value = [];
      recoverableJobItems.value = [];
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
  <v-dialog v-model="viewDialog" v-if="!cleanupTrashcanDialog" max-height="75VH" max-width="950px">
    <v-card title="Backup">
      <v-card-text v-if="isLoading">
        <v-progress-linear indeterminate />
      </v-card-text>
      <v-card-text v-else>
        <v-tabs v-model="tab">
          <v-tab value="details" :rounded="false">Details</v-tab>
          <v-tab value="jobs" :rounded="false">Jobs</v-tab>
          <v-tab value="recovery" :rounded="false" v-if="backup?.status !== BackupStatus.BACKUP_DELETED">Recovery</v-tab>
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
              :items-length="backup?.jobs_total ? backup?.jobs_total : 0"
              :items="jobItems"
              :headers="headers"
              :loading="listIsLoading"
              item-value="job.id"
            >
              <template #[`item.job.status`]="{ item }">
                <v-icon color="warning" v-if="item.job.status === JobStatus.SCHEDULED">mdi-clock-outline</v-icon>
                <v-icon color="success" v-else-if="item.job.status === JobStatus.FINISHED_OK">mdi-check</v-icon>
                <v-icon
                  color="error"
                  v-else-if="
                    item.job.status === JobStatus.ERROR ||
                    item.job.status === JobStatus.FINISHED_ERROR ||
                    item.job.status === JobStatus.FINISHED_QUOTA_ERROR
                  "
                  >mdi-close-circle-outline
                </v-icon>
                <v-icon color="success" v-else-if="item.job.status === JobStatus.JOB_DELETED">mdi-check</v-icon>
                <v-icon color="grey" v-else>mdi-close-circle-outline</v-icon>
              </template>
            </v-data-table-server>
          </v-window-item>
          <v-window-item value="recovery">
            <p class="pa-2 text-button text-center text-info">
              <v-icon size="small">mdi-information-variant-circle-outline</v-icon>
              Recovery commands needs to be executed from entitled principals to recover data
            </p>
            <template v-if="backup?.type === BackupType.CLOUD_STORAGE">
              <v-card>
                <v-card-text>
                  <p>Use the following command to restore your Cloud Storage backup:</p>
                  <v-textarea
                    readonly
                    outlined
                    :model-value="`gcloud transfer jobs create gs://${backup.sink} gs://<TARGET_BUCKET_NAME>`"
                    append-inner-icon="mdi-content-copy"
                    hint="Replace <TARGET_BUCKET_NAME> with your desired target bucket name."
                    persistent-hint
                    @click:append-inner="
                      () => {
                        copyToClipboard(
                          `gcloud transfer jobs create gs://${backup?.sink} gs://<TARGET_BUCKET_NAME>`,
                          notificationsStore.addNotification,
                        );
                      }
                    "
                  >
                  </v-textarea>
                </v-card-text>
              </v-card>
            </template>

            <template v-else-if="backup?.type === BackupType.BIG_QUERY">
              <v-row>
                <v-col>
                  <v-textarea
                    placeholder="Recovery commands will be shown here once you select a job or use the 'Recovery until now' button."
                    readonly
                    outlined
                    :loading="recoveryIsLoading"
                    :model-value="restoreActions.join('\n\n')"
                    append-inner-icon="mdi-content-copy"
                    @click:append-inner="
                      () => {
                        copyToClipboard(restoreActions.join('\n\n'), notificationsStore.addNotification);
                      }
                    "
                  >
                  </v-textarea>
                </v-col>
              </v-row>
              <v-row no-gutters>
                <v-col cols="3">
                  <v-btn variant="outlined" prepend-icon="mdi-code-greater-than-or-equal" @click="loadRestore()">
                    Recovery until now
                  </v-btn>
                </v-col>
                <v-col>
                  <div class=" pl-4 text-subtitle-1">
                    Recovery commands are generated for backup either from the latest job run or at the timestamp of the
                    selected BigQuery job. The latter will omit any backup jobs after the selected timestamp.
                  </div>
                </v-col>
              </v-row>
              <v-data-table-server
                @update:options="loadRecoverableJobs"
                :items-length="backup?.recoverable_jobs_total ? backup?.recoverable_jobs_total : 0"
                :items="recoverableJobItems"
                :headers="[
                  { title: 'Source', key: 'job.source', sortable: false },
                  { title: 'Updated', key: 'job.updated', sortable: false },
                  { title: 'Actions', key: 'action', sortable: false },
                ]"
                :loading="listIsLoading"
                item-value="job.id"
              >
                <template #[`item.action`]="{ item, internalItem, isExpanded }">
                  <v-btn
                    :disabled="isExpanded(internalItem)"
                    variant="outlined"
                    prepend-icon="mdi-code-greater-than-or-equal"
                    @click="loadRestore(item.job.id)"
                  >
                    Recovery until this job
                  </v-btn>
                </template>
              </v-data-table-server>
            </template>

            <template v-else>
              <v-card>
                <v-card-text>
                  <p>Recovery options are not available for this backup type.</p>
                </v-card-text>
              </v-card>
            </template>
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
