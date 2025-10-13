<script setup lang="ts">
import { BackupStatus, CreateRequest, DefaultService } from "@/models/api";
import Notification from "@/models/notification";
import { useNotificationsStore } from "@/stores";
import { computed, ref, watch } from "vue";
import { use } from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { LineChart } from "echarts/charts";
import { TitleComponent, TooltipComponent, LegendComponent, GridComponent } from "echarts/components";
import VChart from "vue-echarts";

use([
  LineChart,
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
  CanvasRenderer,
]);

const notificationsStore = useNotificationsStore();

const props = defineProps<{ backup?: CreateRequest | undefined }>();

const isLoading = ref(false);
const pricePrediction = ref<{ name: string; data: { value: number; name: string }[]; size_in_gb: number }[]>([]);

const chartOptions = ref({
  title: {
    text: "",
  },
  tooltip: {
    trigger: "axis",
    axisPointer: {
      type: "cross",
    },
  },
  legend: {
    show: false,
  },
  grid: {
    left: "3%",
    right: "5%",
    bottom: "5%",
  },
  xAxis: {
    type: "category",
    boundaryGap: false,
    name: "Months",
    nameLocation: "middle",
    nameGap: 30,
    data: [] as string[],
  },
  yAxis: {
    type: "value",
    name: "Costs in €",
    nameLocation: "middle",
    nameGap: 50,
    nameTextStyle: {
      fontWeight: "bold",
      color: "#333",
    },
  },
  series: [] as any[],
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
            data: resp.costs?.map((p) => {
              return { value: Number(p.cost!.toFixed(2)), name: `${p.period}` }
            }) ?? [],
            size_in_gb: size_in_gb,
          },
        ];

        chartOptions.value.series = [{
          name: `€ at month for ${size_in_gb.toFixed(2)} GB`,
          type: "line",
          data: pricePrediction.value[0]?.data ?? [],
        }];

        // Generate xAxis data from API response
        chartOptions.value.xAxis.data = resp.costs?.map((p) => `${p.period}`) ?? [];
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

<template v-if="!backupStatusIsDeleted && (pricePrediction.length > 0 || isLoading)">
  <h4>Cost prediction</h4>
  <v-progress-linear v-if="isLoading" indeterminate />
  <VChart :option="pricePrediction.length === 0 ? {} : chartOptions" :style="{ height: `250px`, width: `400px` }" />
  <small>
    Cost calculation based on current amount of data.
    <b>Additional written data will increase pricing</b>
  </small><br>
  <small v-if="pricePrediction[0]?.size_in_gb">
    Estimate of Network Data Transfer costs for a full backup of {{ pricePrediction[0]?.size_in_gb.toFixed(2) }} GB:
    <!-- This is a rough estimate because network costs are negligible compared to storage costs     -->
    <b v-if="0.02 * pricePrediction[0]?.size_in_gb > 0">{{ (0.02 * pricePrediction[0]?.size_in_gb).toFixed(2) }}€ - {{
      (0.05 * pricePrediction[0].size_in_gb).toFixed(2) }}€</b>
    <b v-else>~0€</b>
  </small>
</template>
