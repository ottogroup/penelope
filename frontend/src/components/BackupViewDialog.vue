<script setup lang="ts">
import { ref, watch } from 'vue'
import { Backup, DefaultService, Job } from "@/models/api";
import { useNotificationsStore } from '@/stores';

const props = defineProps({
    id: {
        type: String,
    },
});

const viewDialog = ref(false);
const isLoading = ref(true);
const item = ref<Backup|undefined>(undefined);
const jobItems = ref<Job[]>([]);

const updateData = () => {
    isLoading.value = true;
    if(props.id) {
        DefaultService.getBackups1(props.id!).then((response) => {
            item.value = response;
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
        key: "status",
    },
    {
        title: "Source",
        key: "source",
    },
    {
        title: "Updated",
        key: "updated",
    },
    {
        title: "Foreign Job ID",
        key: "foreign_job_id",
    },
    {
        title: "Actions",
        key: "action",
    }

]

const loadJobs = ({ page, itemsPerPage, sortBy }: { page: number, itemsPerPage: number, sortBy: string }) => {
    if(page === 1) {
        jobItems.value = item.value?.jobs ?? [];
    }
};

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
            title="Backup details"
        >
        <v-card-text v-if="isLoading">
            <v-progress-linear indeterminate />
        </v-card-text>
        <v-card-text v-else>
            <v-table>
                <tbody>
                    <tr>
                        <td>Created:</td>
                        <td>{{ item?.created }}</td>
                    </tr>
                    <tr>
                        <td>Updated:</td>
                        <td>{{ item?.updated }}</td>
                    </tr>
                    <tr>
                        <td>Storage region:</td>
                        <td>{{ item?.target?.region }}</td>
                    </tr>
                    <tr>
                        <td>Storage class:</td>
                        <td>{{ item?.target?.storage_class }}</td>
                    </tr>
                </tbody>
            </v-table>
            <v-data-table-server
                @update:options="loadJobs"
                :items-length="item?.jobs_total ?? 0"
                :items="jobItems"
                :headers="headers"
            >
            </v-data-table-server>
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