<script setup lang="ts">
import { DefaultService, Backup, UpdateRequest } from '@/models/api';
import Notification from '@/models/notification';
import { usePrincipalStore, useNotificationsStore } from '@/stores';
import { ref, watch } from 'vue'

const principalStore = usePrincipalStore();
const notificationsStore = useNotificationsStore();

const props = defineProps({
    id: {
        type: String,
    },
});

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
    snapshot_options: {
    },
});

const isValid = ref(false);

const updateData = () => {
    isLoading.value = true;
    if(props.id) {
        DefaultService.getBackups1(props.id!).then((response) => {
            backup.value = response;
        }).catch((err) => {
            useNotificationsStore().handleError(err);
        }).finally(() => {
            isLoading.value = false;
        });
    }
}

const saveBackup = () => {
    if (!isValid.value) {
        return;
    }
    isLoading.value = true;
    const req: UpdateRequest = {
        backup_id: props.id!,
        mirror_ttl: backup.value.snapshot_options?.lifetime_in_days,
        snapshot_ttl: backup.value.snapshot_options?.lifetime_in_days,
        archive_ttm: backup.value.target?.archive_ttm,
        include_path: backup.value.gcs_options?.include_prefixes,
        exclude_path: backup.value.gcs_options?.exclude_prefixes,
        table: backup.value.bigquery_options?.table,
        excluded_tables: backup.value.bigquery_options?.excluded_tables,
    };

    DefaultService.patchBackups(req).then(() => {
        notificationsStore.addNotification(new Notification({
            message: 'Backup updated',
            color: 'success',
        }));
        viewDialog.value = false;
    }).catch((err) => notificationsStore.handleError(err))
    .finally(() => {
        isLoading.value = false;
    })
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
            title="Update backup"
            :loading="isLoading"
        >
        <v-card-text>
            <v-form :disabled="isLoading" v-model="isValid" fast-fail @submit.prevent>
                <v-row>
                    <v-col>
                        <h3>Source</h3>
                        <template v-if="backup?.type == 'CloudStorage'">
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
                        <template v-if="backup?.type == 'BigQuery'">
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
                        <template v-if="backup?.strategy == 'Snapshot'">
                            <v-text-field
                            label="Snapshot TTL"
                            type="number"
                            hint="After X days data will be deleted. Default is 0."
                            v-model="backup!.snapshot_options!.lifetime_in_days"
                        ></v-text-field>
                        </template>
                        <template v-if="backup?.strategy == 'Oneshot'">
                            <v-text-field
                                label="Oneshot TTL"
                                type="number"
                                hint="After X days data will be deleted. Default is 0."
                                v-model="backup!.snapshot_options!.lifetime_in_days"
                            ></v-text-field>
                        </template>
                        <template v-if="backup?.strategy == 'Mirror'">
                            <v-text-field
                                label="Mirror TTL"
                                type="number"
                                hint="After X days data will be deleted. Default is 0."
                                v-model="backup!.snapshot_options!.lifetime_in_days"
                            ></v-text-field>
                        </template>
                    </v-col>
                </v-row>
            </v-form>
        </v-card-text>
        <template v-slot:actions>
            <v-btn-group class="ms-auto">
                <v-btn
                    text="Cancel"
                    @click="viewDialog = false"
                ></v-btn>
                <v-btn
                    text="Update"
                    :disabled="isLoading || !isValid"
                    @click="saveBackup()"
                ></v-btn>
            </v-btn-group>
        </template>
        </v-card>
    </v-dialog>
</template>