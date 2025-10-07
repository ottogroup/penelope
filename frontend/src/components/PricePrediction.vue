<script setup lang="ts">
import { BackupStatus, CreateRequest, DefaultService } from "@/models/api";
import Notification from "@/models/notification";
import { useNotificationsStore } from "@/stores";
import { ApexOptions } from "apexcharts";
import { computed, ref, watch } from "vue";
import VueApexCharts from "vue3-apexcharts";

const notificationsStore = useNotificationsStore();

const props = defineProps<{ backup: CreateRequest | undefined }>();

const isLoading = ref(false);
const pricePrediction = ref<{ name: string; data: number[]; size_in_gb: number }[]>([]);
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
  return (
    props.backup?.status === BackupStatus.FINISHED ||
    props.backup?.status === BackupStatus.BACKUP_DELETED ||
    props.backup?.status === BackupStatus.BACKUP_SOURCE_DELETED
  );
});

const updateData = () => {
  if (props.backup === undefined) {
    isLoading.value = false;
    pricePrediction.value = [];
    return;
  }
  isLoading.value = true;
  pricePrediction.value = [];

  // network egress costs are not calculated here, bases on
  // https://cloud.google.com/storage/pricing
  // https://cloud.google.com/bigquery/pricing
  if (!backupStatusIsDeleted.value) {
    DefaultService.postBackupsCalculate(props.backup)
      .then((resp) => {
        let size_in_bytes = resp.costs?.[0]?.size_in_bytes ?? 0;
        let size_in_gb = (size_in_bytes / Math.pow(2, 30));
        pricePrediction.value = [
          {
            name: `€ at month for ${size_in_gb.toFixed(2)} GB`,
            data: resp.costs?.map((c) => c.cost!) ?? [],
            size_in_gb: size_in_gb,
          },
        ];
      })
      .catch(() => {
        notificationsStore.addNotification(
          new Notification({
            message: `Could not make cost prediction for backup`,
            color: "warning",
          }),
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
    <v-progress-linear v-if="isLoading" indeterminate />
    <VueApexCharts type="area" :options="pricePredictionOptions" :series="pricePrediction" />
    <small>
      Cost calculation based on current amount of data.
      <b>Additional written data will increase pricing</b>
    </small><br>
    <small v-if="pricePrediction[0]?.size_in_gb">
      Estimate of Network Data Transfer costs for a full backup of {{ pricePrediction[0]?.size_in_gb.toFixed(2) }} GB:
<!-- This is a rough estimate because network costs are negligible compared to storage costs     -->
      <b v-if="0.02 * pricePrediction[0]?.size_in_gb > 0">{{ (0.02 * pricePrediction[0]?.size_in_gb).toFixed(2) }}€ - {{ (0.05 * pricePrediction[0].size_in_gb).toFixed(2) }}€</b>
      <b v-else>~0€</b>
    </small>
  </template>
</template>
