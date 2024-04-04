<script setup lang="ts">
import { ref, watch } from 'vue'
import { Backup, CreateRequest, DefaultService, Job } from "@/models/api";
import { useNotificationsStore } from '@/stores';
import { RestoreResponse } from '@/models/api/models/RestoreResponse';
import PricePrediction from '@/components/PricePrediction.vue';
import ComplianceCheck from '@/components/ComplianceCheck.vue';

const props = defineProps({
    id: {
        type: String,
    },
});

const tab = ref();
const viewDialog = ref(false);
const isLoading = ref(true);
const listIsLoading = ref(true);
const backup = ref<Backup|undefined>(undefined);
const backupForEval = ref<CreateRequest|undefined>(undefined);
const jobItems = ref<{job: Job, restore: RestoreResponse|undefined}[]>([]);

const updateData = () => {
    isLoading.value = true;
    if(props.id) {
        DefaultService.getBackups1(props.id!).then((response) => {
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
            };
        }).catch((err) => {
            useNotificationsStore().handleError(err);
        }).finally(() => {
            isLoading.value = false;
        });
    }
}

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
    }

]

const loadJobs = ({ page, itemsPerPage }: { page: number, itemsPerPage: number, sortBy: string }) => {
    listIsLoading.value = true
    if(page === 1) {
        jobItems.value = backup.value?.jobs?.slice(0, itemsPerPage).map((j: Job) => {return {job: j, restore: undefined}}) ?? [];
        listIsLoading.value = false;
    } else {
        DefaultService.getBackups1(props.id!, itemsPerPage, page).then(
            (resp) => {
                jobItems.value = resp.jobs?.map((j: Job) => {return {job: j, restore: undefined}}) ?? []
            }
        ).catch((err) => {
            useNotificationsStore().handleError(err);
        }).finally(() => {
            listIsLoading.value = false;
        });
    }
};

const loadRestore = (item: {job: Job, restore: RestoreResponse|undefined}) => {
    DefaultService.getRestore(backup.value?.id!, item.job.id).then((resp) => {
        item.restore = resp;
    }).catch((err) => {
        useNotificationsStore().handleError(err);
    });
}

const cloudStorageLink = (project: string, bucket: string) => {
    return `https://console.cloud.google.com/storage/browser/${bucket}?project=${project}`;
}

const bigqueryDatasetLink = (project: string, dataset: string) => {
    return `https://console.cloud.google.com/bigquery?project=${project}&p=${project}&d=${dataset}&page=dataset`;
}

const bigqueryTableLink = (project: string, dataset: string, table: string) => {
    return `https://console.cloud.google.com/bigquery?project=${project}&p=${project}&d=${dataset}&t=${table}&page=table`;
}

watch(() => props.id, (id) => {
    viewDialog.value = !!id;
    updateData();
});
</script>

<template>
    <v-dialog
        v-model="viewDialog"
        width="800"
        >
        <v-card
            title="Backup"
        >
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
                                <td>Project:</td>
                                <td>{{ backup?.project }}</td>
                            </tr>
                            <tr>
                                <td>Type:</td>
                                <td>{{ backup?.type }}</td>
                            </tr>
                            <tr>
                                <td>Source:</td>
                                <td>
                                    <template v-if="backup?.type === 'BigQuery'">
                                        BigQuery: <a :href="bigqueryDatasetLink(backup?.project ?? '', backup?.bigquery_options?.dataset ?? '')">{{ backup?.bigquery_options?.dataset }}</a>
                                        
                                    </template>
                                    <template v-if="backup?.type === 'CloudStorage'">
                                        Bucket: <a :href="cloudStorageLink(backup?.project ?? '', backup?.gcs_options?.bucket ?? '')">{{ backup?.gcs_options?.bucket }}</a>
                                    </template>
                                </td>
                            </tr>
                            <tr v-if="backup?.type === 'BigQuery' && (backup?.bigquery_options?.table?.length ?? 0 > 0)">
                                <td>Tables:</td>
                                <td>
                                    <ul>
                                        <li v-for="table in backup?.bigquery_options?.table">
                                            Table: <a :href="bigqueryTableLink(backup?.project ?? '', backup?.bigquery_options?.dataset ?? '', table)">{{ table }}</a>
                                        </li>
                                    </ul>
                                </td>
                            </tr>
                            <tr v-if="backup?.type === 'BigQuery' && (backup?.bigquery_options?.excluded_tables?.length ?? 0 > 0)">
                                <td>Excluded tables:</td>
                                <td>
                                    <ul>
                                        <li v-for="table in backup?.bigquery_options?.excluded_tables">
                                            Table: <a :href="bigqueryTableLink(backup?.project ?? '', backup?.bigquery_options?.dataset ?? '', table)">{{ table }}</a>
                                        </li>
                                    </ul>
                                </td>
                            </tr>
                            <tr v-if="backup?.type === 'CloudStorage' && (backup?.gcs_options?.include_prefixes?.length ?? 0 > 0)">
                                <td>Included prefixes:</td>
                                <td>
                                    <ul>
                                        <li v-for="prefix in backup?.gcs_options?.include_prefixes">
                                            {{ prefix }}
                                        </li>
                                    </ul>
                                </td>
                            </tr>
                            <tr v-if="backup?.type === 'CloudStorage' && (backup?.gcs_options?.exclude_prefixes?.length ?? 0 > 0)">
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
                                <td colspan="2"><h4>Target</h4></td>
                            </tr>
                            <tr>
                                <td>Sink bucket:</td>
                                <td>{{ backup?.sink }}</td>
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
                            <v-icon v-if="item.job.status === 'Scheduled'">mdi-clock-outline</v-icon>
                            <v-icon v-if="item.job.status === 'FinishedOk'">mdi-check</v-icon>
                            <v-icon v-else>mdi-close-circle-outline</v-icon>
                            <!-- // TODO: handle other job statuses -->
                        </template>
                        <template #[`item.action`]="{ item, internalItem, toggleExpand, isExpanded }">
                            <v-btn v-if="backup?.type === 'BigQuery' && !isExpanded(internalItem)" variant="outlined" @click="loadRestore(item); toggleExpand(internalItem)">
                                show import bq
                            </v-btn>
                        </template>
                        <template v-slot:expanded-row="{ columns, item }">
                            <tr>
                                <td :colspan="columns.length">
                                    <span v-for="action in item.restore?.actions">
                                        {{ action.action }}
                                        <br>
                                    </span>
                                </td>
                            </tr>
                        </template>
                    </v-data-table-server>
                </v-window-item>
            </v-window>
        </v-card-text>
        <template v-slot:actions>
            <v-btn
                class="ms-auto"
                text="Close"
                @click="viewDialog = false"
            ></v-btn>
        </template>
        </v-card>
    </v-dialog>
</template>