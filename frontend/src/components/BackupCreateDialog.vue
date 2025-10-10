<script setup lang="ts">
import ComplianceCheck from "@/components/ComplianceCheck.vue";
import PricePrediction from "@/components/PricePrediction.vue";
import { capitalize } from "@/helpers/filters";
import { BackupStrategy, DefaultService, SourceProject } from "@/models/api";
import { BackupType } from "@/models/api/models/BackupType";
import { CreateRequest } from "@/models/api/models/CreateRequest";
import Notification from "@/models/notification";
import { useNotificationsStore, usePrincipalStore } from "@/stores";
import { ref, watch } from "vue";

const principalStore = usePrincipalStore();
const notificationsStore = useNotificationsStore();

const model = defineModel<boolean>();

const isLoading = ref(true);
const sourceProjects = ref<string[]>([]);
const sourceProject = ref<SourceProject>();
const storageClasses = ref<{ title: string; value: string }[]>([]);
const storageRegions = ref<string[]>([]);
const backupTypes = ref([
  { title: "Cloud Storage", value: BackupType.CLOUD_STORAGE },
  { title: "BigQuery", value: BackupType.BIG_QUERY },
]);
const strategies = ref(Object.values(BackupStrategy));

const loadingSourceProject = ref(false);
const loadingBucketNames = ref(false);
const bucketNames = ref<string[]>([]);
const loadingDatasetNames = ref(false);
const datasetNames = ref<string[]>([]);

const request = ref<CreateRequest>({
  gcs_options: {},
  bigquery_options: {},
  target: {},
  snapshot_options: {},
});
const isValid = ref(false);

const evalutingBackup = ref<CreateRequest | undefined>();

const updateData = async () => {
  request.value = {
    gcs_options: {},
    bigquery_options: {},
    target: {},
    snapshot_options: {},
  };
  evalutingBackup.value = undefined;
  isLoading.value = true;
  sourceProjects.value = principalStore.principal.getProjects();
  Promise.all([DefaultService.getConfigRegions(), DefaultService.getConfigStorageClasses()])
    .then(([regionResponse, storageClassResponse]) => {
      storageClasses.value =
        storageClassResponse.storage_classes
          ?.map((c) => {
            return {
              title: capitalize(c.toLowerCase()),
              value: c,
            };
          })
          .sort() ?? [];
      storageRegions.value = regionResponse.regions?.sort() ?? [];
    })
    .catch((err) => notificationsStore.handleError(err))
    .finally(() => {
      isLoading.value = false;
    });
};

const updateBucketNames = () => {
  bucketNames.value = [];
  if (request.value.project) {
    loadingBucketNames.value = true;
    DefaultService.getBuckets(request.value.project)
      .then((response) => {
        bucketNames.value = response.buckets ?? [];
      })
      .catch((err) => notificationsStore.handleError(err))
      .finally(() => {
        loadingBucketNames.value = false;
      });
  }
};

const updateDatasetNames = () => {
  bucketNames.value = [];
  if (request.value.project) {
    loadingDatasetNames.value = true;
    DefaultService.getDatasets(request.value.project)
      .then((response) => {
        datasetNames.value = response.datasets ?? [];
      })
      .catch((err) => notificationsStore.handleError(err))
      .finally(() => {
        loadingDatasetNames.value = false;
      });
  }
};

function updateSourceProject() {
  if (request.value.project) {
    loadingSourceProject.value = true;
    DefaultService.getSourceProject(request.value.project)
      .then((response) => {
        sourceProject.value = response.source_project ?? {};
      })
      .catch((err) => notificationsStore.handleError(err))
      .finally(() => {
        loadingSourceProject.value = false;
      });
  }
}

const updateSourceFields = () => {
  if (request.value.type == BackupType.CLOUD_STORAGE) {
    updateBucketNames();
  } else if (request.value.type == BackupType.BIG_QUERY) {
    updateDatasetNames();
  }
  updateSourceProject();
};

const apiRequestBody = () => {
  const req: CreateRequest = {
    project: request.value.project,
    recovery_point_objective: Number(request.value.recovery_point_objective),
    recovery_time_objective: Number(request.value.recovery_time_objective),
    type: request.value.type,
    strategy: request.value.strategy,
    target: {
      storage_class: request.value.target?.storage_class,
      region: request.value.target?.region,
      dual_region: request.value.target?.dual_region,
      archive_ttm: Number(request.value.target?.archive_ttm),
    },
    snapshot_options: {
      lifetime_in_days: Number(request.value.snapshot_options?.lifetime_in_days),
      frequency_in_hours: Number(request.value.snapshot_options?.frequency_in_hours),
      last_scheduled: request.value.snapshot_options?.last_scheduled,
    },
  };
  if (request.value.type == BackupType.CLOUD_STORAGE) {
    req.gcs_options = request.value.gcs_options;
  } else if (request.value.type == BackupType.BIG_QUERY) {
    req.bigquery_options = request.value.bigquery_options;
  }

  if (request.value.strategy == BackupStrategy.ONESHOT) {
    req.strategy = BackupStrategy.SNAPSHOT;
    req.snapshot_options!.frequency_in_hours = 0;
  }
  return req;
};

const saveBackup = () => {
  if (!isValid.value) {
    return;
  }
  isLoading.value = true;
  const req = apiRequestBody();

  DefaultService.postBackups(req)
    .then(() => {
      notificationsStore.addNotification(
        new Notification({
          message: "Backup created",
          color: "success",
        }),
      );
      model.value = false;
    })
    .catch((err) => notificationsStore.handleError(err))
    .finally(() => {
      isLoading.value = false;
    });
};

const requiredRule = (fieldName: string) => {
  return (v: string) => (!!v && v.length > 0) || `${fieldName} is required`;
};

const integerRequiredRule = (fieldName: string) => {
  return (v: number) => (!!v && v > 0) || `${fieldName} is required and must be bigger than 0`;
};

watch(
  () => model.value,
  (value) => {
    if (value) {
      updateData();
    }
  },
);
</script>

<template>
  <v-dialog v-model="model" width="800">
    <v-card title="Create backup" :loading="isLoading">
      <v-card-text>
        <v-form :disabled="isLoading" v-model="isValid" fast-fail @submit.prevent>
          <v-row>
            <v-col>
              <h3>Source</h3>
              <v-select
                class="mb-2"
                label="Project*"
                :items="sourceProjects"
                v-model="request.project"
                @update:model-value="updateSourceFields()"
                :rules="[requiredRule('Project')]"
              ></v-select>
              <v-text-field
                class="mb-2"
                v-if="sourceProject"
                label="Data owner"
                v-model="sourceProject.data_owner"
                readonly
              ></v-text-field>
              <v-text-field
                class="mb-2"
                v-if="sourceProject"
                label="Availability class"
                v-model="sourceProject.availability_class"
                readonly
              ></v-text-field>
              <v-select
                class="mb-2"
                label="Backup type*"
                :items="backupTypes"
                v-model="request.type"
                @update:model-value="updateSourceFields()"
                :rules="[requiredRule('Backup type')]"
              ></v-select>
              <v-text-field
                class="mb-2"
                label="RPO (hours)*"
                type="number"
                hint="Recovery time objective: Minimal frequency a backup must be conducted."
                v-model="request.recovery_point_objective"
                :rules="[integerRequiredRule('Recovery point objective (hours)')]"
              ></v-text-field>
              <v-text-field
                class="mb-2"
                label="RTO (minutes)*"
                type="number"
                hint="Recovery time objective: The recovery process time duration needed to restore data from backup storage."
                :rules="[integerRequiredRule('Recovery time objective (minutes)')]"
                v-model="request.recovery_time_objective"
              ></v-text-field>
              <template v-if="request.type == BackupType.CLOUD_STORAGE">
                <v-select
                  class="mb-2"
                  label="Bucket name*"
                  :items="bucketNames"
                  :loading="loadingBucketNames"
                  v-model="request.gcs_options!.bucket"
                ></v-select>
                <v-combobox
                  class="mb-2"
                  chips
                  multiple
                  clearable
                  label="Include paths"
                  v-model="request.gcs_options!.include_prefixes"
                ></v-combobox>
                <v-combobox
                  class="mb-2"
                  chips
                  multiple
                  clearable
                  label="Exclude paths"
                  v-model="request.gcs_options!.exclude_prefixes"
                ></v-combobox>
              </template>
              <template v-if="request.type == BackupType.BIG_QUERY">
                <v-select
                  class="mb-2"
                  label="Dataset*"
                  :items="datasetNames"
                  :loading="loadingDatasetNames"
                  v-model="request.bigquery_options!.dataset"
                ></v-select>
                <v-combobox
                  class="mb-2"
                  chips
                  multiple
                  clearable
                  label="BigQuery tables"
                  hint="When empty will take all tables."
                  v-model="request.bigquery_options!.table"
                ></v-combobox>
                <v-combobox
                  class="mb-2"
                  chips
                  multiple
                  clearable
                  label="BigQuery excluded tables"
                  hint="When present will ignore given tables."
                  v-model="request.bigquery_options!.excluded_tables"
                ></v-combobox>
              </template>
            </v-col>
            <v-col>
              <h3>Target</h3>
              <v-select
                class="mb-2"
                label="Storage class*"
                :items="storageClasses"
                hint="Bucket storage class for data"
                v-model="request.target!.storage_class"
                :rules="[requiredRule('Storage class')]"
              ></v-select>
              <v-select
                class="mb-2"
                label="Storage region*"
                :items="storageRegions"
                v-model="request.target!.region"
                :rules="[requiredRule('Storage region')]"
              ></v-select>
              <v-select
                class="mb-2"
                label="Secondary storage region"
                :items="storageRegions"
                clearable
                v-model="request.target!.dual_region"
              ></v-select>
              <v-text-field
                class="mb-2"
                label="Archive transition"
                type="number"
                hint="After X days change object storage class to archive. Default is 0."
                v-model="request.target!.archive_ttm"
              ></v-text-field>
            </v-col>
            <v-col>
              <h3>Details</h3>
              <v-select
                class="mb-2"
                label="Strategy*"
                :items="strategies"
                v-model="request.strategy"
                hint="Snapshot: one or many shots. Mirror: hourly sync."
                :rules="[requiredRule('Strategy')]"
              ></v-select>
              <template v-if="request.strategy == BackupStrategy.SNAPSHOT">
                <v-text-field
                  class="mb-2"
                  label="Snapshot TTL"
                  type="number"
                  hint="After X days data will be deleted. Default is 0."
                  v-model="request.snapshot_options!.lifetime_in_days"
                ></v-text-field>
                <v-text-field
                  class="mb-2"
                  label="Snapshot schedule*"
                  type="number"
                  hint="Snapshot will be created every X hours at full hour"
                  v-model="request.snapshot_options!.frequency_in_hours"
                  :rules="[
                    (v) => {
                      if (request.strategy === BackupStrategy.SNAPSHOT) {
                        return (!!v && Number(v) > 0) || 'Snapshot schedule must be greater than 0';
                      }
                      return true;
                    },
                  ]"
                ></v-text-field>
              </template>
              <template v-if="request.strategy == 'Oneshot'">
                <v-text-field
                  class="mb-2"
                  label="Oneshot TTL"
                  type="number"
                  hint="After X days data will be deleted. Default is 0."
                  v-model="request.snapshot_options!.lifetime_in_days"
                ></v-text-field>
              </template>
              <template v-if="request.strategy == BackupStrategy.MIRROR">
                <v-text-field
                  class="mb-2"
                  label="Mirror TTL"
                  type="number"
                  hint="After X days data will be deleted. Default is 0."
                  v-model="request.snapshot_options!.lifetime_in_days"
                ></v-text-field>
              </template>
            </v-col>
          </v-row>
        </v-form>
        <v-row v-if="evalutingBackup">
          <v-col>
            <PricePrediction :backup="evalutingBackup" />
          </v-col>
          <v-col>
            <ComplianceCheck :backup="evalutingBackup" />
          </v-col>
        </v-row>
      </v-card-text>
      <template v-slot:actions>
        <v-btn-group class="ms-auto">
          <v-btn text="Cancel" @click="model = false"></v-btn>
          <v-btn text="Create" :disabled="isLoading || !isValid" @click="saveBackup()"></v-btn>
          <v-btn text="Evaluate" :disabled="isLoading || !isValid" @click="evalutingBackup = apiRequestBody()"></v-btn>
        </v-btn-group>
      </template>
    </v-card>
  </v-dialog>
</template>
