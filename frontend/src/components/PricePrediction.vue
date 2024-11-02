<script setup lang="ts">
import {BackupStatus, CreateRequest, DefaultService} from "@/models/api";
import {useNotificationsStore} from "@/stores";
import {ApexOptions} from "apexcharts";
import {computed, ref, watch} from "vue";
import VueApexCharts from "vue3-apexcharts";
import Notification from "@/models/notification";

const notificationsStore = useNotificationsStore();

const props = defineProps<{ backup: CreateRequest | undefined }>();

const isLoading = ref(false);
const pricePrediction = ref<{ name: string; data: number[] }[]>([]);
const pricePredictionOptions = ref<ApexOptions>({
  yaxis: {
    title: {
      text: "Price",
    },
    min: 0,
    decimalsInFloat: 2,
  },
  stroke: {
    curve: "straight",
  },
  chart: {
    animations: {
      enabled: false,
    },
    toolbar: {
      show: false,
      tools: {
        download: false,
      },
    },
  },
  grid: {
    row: {
      colors: ["#f3f3f3"],
      opacity: 0.5,
    },
  },
  dataLabels: {
    enabled: false,
  },
  legend: {
    show: true,
    showForSingleSeries: true,
  },
});

const backupStatusIsDeleted = computed(() => {
  return props.backup?.status === BackupStatus.FINISHED || props.backup?.status === BackupStatus.BACKUP_DELETED || props.backup?.status === BackupStatus.BACKUP_SOURCE_DELETED
});

const updateData = () => {
  if (props.backup === undefined) {
    isLoading.value = false;
    pricePrediction.value = [];
    return;
  }
  isLoading.value = true;
  pricePrediction.value = [];


  if (!backupStatusIsDeleted.value) {
    DefaultService.postBackupsCalculate(props.backup)
      .then((resp) => {
        pricePrediction.value = [
          {
            name: `€ at month for ${((resp.costs?.[0]?.size_in_bytes ?? 0) / Math.pow(2, 30)).toFixed(2)} GB`,
            data: resp.costs?.map((c) => c.cost!) ?? [],
          },
        ];
      })
      .catch(() => {
        notificationsStore.addNotification(
          new Notification({
            message: `Could not make cost prediction for backup`,
            color: "warning",
          })
        );
      })
      .finally(() => {
        isLoading.value = false;
      });
  }
};

updateData();
watch(
  () => props.backup,
  () => {
    updateData();
  },
);
</script>

<template>
  <template v-if="!backupStatusIsDeleted && (pricePrediction.length > 0 || isLoading)">
    <h4>Cost prediction</h4>
    <v-progress-linear v-if="isLoading" indeterminate/>
    <VueApexCharts type="area" :options="pricePredictionOptions" :series="pricePrediction"/>
    <small
    >* cost calculation based on current amount of data. <b>Additional written data will increase pricing</b></small
    >
  </template>
</template>
