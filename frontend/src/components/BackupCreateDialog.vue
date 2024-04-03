<script setup lang="ts">
import { capitalize } from '@/helpers/filters';
import { DefaultService } from '@/models/api';
import { CreateRequest } from '@/models/api/models/CreateRequest';
import Notification from '@/models/notification';
import { usePrincipalStore, useNotificationsStore } from '@/stores';
import { ref, watch } from 'vue'
import PricePrediction from '@/components/PricePrediction.vue';
import ComplianceCheck from '@/components/ComplianceCheck.vue';


const principalStore = usePrincipalStore();
const notificationsStore = useNotificationsStore();

const model = defineModel<boolean>();

const isLoading = ref(true);
const sourceProjects = ref<string[]>([]);
const storageClasses = ref<{title: string, value: string}[]>([]);
const storageRegions = ref<string[]>([]);
const backupTypes = ref([{title: 'Cloud Storage', value: 'CloudStorage'}, {title: 'BigQuery', value: 'BigQuery'}]);
const strategies = ref(['Snapshot', 'Oneshot', 'Mirror']);

const loadingBucketNames = ref(false);
const bucketNames = ref<string[]>([]);
const loadingDatasetNames = ref(false);
const datasetNames = ref<string[]>([]);

const request = ref<CreateRequest>({
    gcs_options: {},
    bigquery_options: {},
    target: {},
    snapshot_options: {},
})
const isValid = ref(false);

const evalutingBackup = ref<CreateRequest|undefined>();

const updateData = async () => {
    request.value = {
        gcs_options: {},
        bigquery_options: {},
        target: {},
        snapshot_options: {},
    };
    evalutingBackup.value = undefined;
    isLoading.value = true;
    sourceProjects.value = principalStore.principal.getProjects()
    Promise.all([
        DefaultService.getConfigRegions(),
        DefaultService.getConfigStorageClasses(),
    ]).then(([regionResponse, storageClassResponse]) => {
        storageClasses.value = storageClassResponse.storage_classes?.map((c) => { return {
            title: capitalize(c.toLowerCase()),
            value: c,
        }}) ?? [];
        storageRegions.value = regionResponse.regions ?? [];
    }).catch((err) => notificationsStore.handleError(err)).finally(() => {
        isLoading.value = false;
    });
}

const updateBucketNames = () => {
    bucketNames.value = [];
    if(request.value.project) {
        loadingBucketNames.value = true;
        DefaultService.getBuckets(request.value.project).then((response) => {
            bucketNames.value = response.buckets ?? [];
        }).catch((err) => notificationsStore.handleError(err))
        .finally(() => {
            loadingBucketNames.value = false;
        })
    }
}

const updateDatasetNames = () => {
    bucketNames.value = [];
    if(request.value.project) {
        loadingDatasetNames.value = true;
        DefaultService.getDatasets(request.value.project).then((response) => {
            datasetNames.value = response.datasets ?? [];
        }).catch((err) => notificationsStore.handleError(err))
        .finally(() => {
            loadingDatasetNames.value = false;
        })
    }
}

const updateSourceFields = () => {
    if(request.value.type == 'CloudStorage') {
        updateBucketNames();
    } else if(request.value.type == 'BigQuery') {
        updateDatasetNames();
    }
}

const apiRequestBody = () => {
    const req: CreateRequest = {
        project: request.value.project,
        type: request.value.type,
        strategy: request.value.strategy,
        target: request.value.target,
        snapshot_options: {
            lifetime_in_days: Number(request.value.snapshot_options?.lifetime_in_days),
            frequency_in_hours: Number(request.value.snapshot_options?.frequency_in_hours),
            last_scheduled: request.value.snapshot_options?.last_scheduled,
        },
    }
    if(request.value.type == 'CloudStorage') {
        req.gcs_options = request.value.gcs_options;
    } else if(request.value.type == 'BigQuery') {
        req.bigquery_options = request.value.bigquery_options;
    }

    if (request.value.strategy == 'Oneshot') {
        req.strategy = 'Snapshot'
        req.snapshot_options!.frequency_in_hours = 0
    }
    return req;
}

const saveBackup = () => {
    if (!isValid.value) {
        return;
    }
    isLoading.value = true;
    const req = apiRequestBody();

    DefaultService.postBackups(req).then(() => {
        notificationsStore.addNotification(new Notification({
            message: 'Backup created',
            color: 'success',
        }));
        model.value = false;
    }).catch((err) => notificationsStore.handleError(err))
    .finally(() => {
        isLoading.value = false;
    })
}

const requiredRule = (fieldName: string) => {
  return (v: string) => (!!v && v.length > 0) || `${fieldName} is required`;
};

watch(
    () => model.value, 
    (value) => {
        if(value) {
            updateData();
        }
    }
)
</script>

<template>
    <v-dialog
        v-model="model"
        width="800"
        >
        <v-card
            title="Create backup"
            :loading="isLoading"
        >
        <v-card-text>
            <v-form :disabled="isLoading" v-model="isValid" fast-fail @submit.prevent>
                <v-row>
                    <v-col>
                        <h3>Source</h3>
                        <v-select
                            label="Project*"
                            :items="sourceProjects"
                            v-model="request.project"
                            @update:model-value="updateSourceFields()"
                            :rules="[requiredRule('Project')]"
                        ></v-select>
                        <v-select
                            label="Backup type*"
                            :items="backupTypes"
                            v-model="request.type"
                            @update:model-value="updateSourceFields()"
                            :rules="[requiredRule('Backup type')]"
                        ></v-select>
                        <template v-if="request.type == 'CloudStorage'">
                            <v-select
                                label="Bucket name*"
                                :items="bucketNames"
                                :loading="loadingBucketNames"
                                v-model="request.gcs_options!.bucket"
                            ></v-select>
                            <v-combobox
                                chips
                                multiple
                                clearable
                                label="Include paths"
                                v-model="request.gcs_options!.include_prefixes"
                            ></v-combobox>
                            <v-combobox
                                chips
                                multiple
                                clearable
                                label="Exclude paths"
                                v-model="request.gcs_options!.exclude_prefixes"
                            ></v-combobox>
                        </template>
                        <template v-if="request.type == 'BigQuery'">
                            <v-select
                                label="Dataset*"
                                :items="datasetNames"
                                :loading="loadingDatasetNames"
                                v-model="request.bigquery_options!.dataset"
                            ></v-select>
                            <v-combobox
                                chips
                                multiple
                                clearable
                                label="BigQuery tables"
                                hint="When empty will take all tables."
                                v-model="request.bigquery_options!.table"
                            ></v-combobox>
                            <v-combobox
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
                            label="Storage class*"
                            :items="storageClasses"
                            hint="Bucket storage class for data"
                            v-model="request.target!.storage_class"
                            :rules="[requiredRule('Storage class')]"
                        ></v-select>
                        <v-select
                            label="Storage region*"
                            :items="storageRegions"
                            v-model="request.target!.region"
                            :rules="[requiredRule('Storage region')]"
                        ></v-select>
                        <v-select
                            label="Secondary storage region"
                            :items="storageRegions"
                            clearable
                            v-model="request.target!.dual_region"
                        ></v-select>
                        <v-text-field
                            label="Archive TTM"
                            type="number"
                            hint="After X days change object storage class to archive. Default is 0."
                            v-model="request.target!.archive_ttm"
                        ></v-text-field>
                    </v-col>
                    <v-col>
                        <h3>Details</h3>
                        <v-select
                            label="Strategy*"
                            :items="strategies"
                            v-model="request.strategy"
                            hint="Snapshot: one or many shots. Mirror: hourly sync."
                            :rules="[requiredRule('Strategy')]"
                        ></v-select>
                        <template v-if="request.strategy == 'Snapshot'">
                            <v-text-field
                            label="Snapshot TTL"
                            type="number"
                            hint="After X days data will be deleted. Default is 0."
                            v-model="request.snapshot_options!.lifetime_in_days"
                        ></v-text-field>
                        <v-text-field
                            label="Snapshot schedule"
                            type="number"
                            hint="Snapshot will be created every X hours at full hour"
                            v-model="request.snapshot_options!.frequency_in_hours"
                        ></v-text-field>
                        </template>
                        <template v-if="request.strategy == 'Oneshot'">
                            <v-text-field
                                label="Oneshot TTL"
                                type="number"
                                hint="After X days data will be deleted. Default is 0."
                                v-model="request.snapshot_options!.lifetime_in_days"
                            ></v-text-field>
                        </template>
                        <template v-if="request.strategy == 'Mirror'">
                            <v-text-field
                                label="Mirror TTL"
                                type="number"
                                hint="After X days data will be deleted. Default is 0."
                                v-model="request.snapshot_options!.lifetime_in_days"
                            ></v-text-field>
                        </template>
                    </v-col>
                </v-row>
            </v-form>
            <v-row>
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
                <v-btn
                    text="Cancel"
                    @click="model = false"
                ></v-btn>
                <v-btn
                    text="Create"
                    :disabled="isLoading || !isValid"
                    @click="saveBackup()"
                ></v-btn>
                <v-btn
                    text="Evaluate"
                    :disabled="isLoading || !isValid"
                    @click="evalutingBackup = apiRequestBody();"
                ></v-btn>
            </v-btn-group>
        </template>
        </v-card>
    </v-dialog>
</template>