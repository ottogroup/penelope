<script setup lang="ts">
import { useNavigate } from "@/router/navigate";
import { usePrincipalStore, useRouteStore } from "@/stores";
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";

let loginWaitDialog = ref(true);
let loginFailed = ref(false);

const retryLogin = () => location.reload();

onMounted(() => {
  const principalStore = usePrincipalStore();
  const router = useRouter();
  const navigateTo = useNavigate();
  const routerStore = useRouteStore();

  principalStore.principal
    .initPrincipal()
    .then(() => {
      loginWaitDialog.value = false;
      console.log("User logged in. Redirecting");
      if (routerStore.returnUrl == "") {
        routerStore.returnUrl = "/";
      }
      navigateTo(router, { path: routerStore.returnUrl }).then(() => principalStore.finalizePrincipal());
    })
    .catch(() => (loginFailed.value = true));
});
</script>

<template>
  <v-container :fluid="true">
    <v-row justify="center">
      <v-dialog v-model="loginWaitDialog" :persistent="true" max-width="290">
        <v-card v-if="loginFailed">
          <v-card-title class="headline"> Login failed</v-card-title>
          <v-card-text class="pa-2"> Error while obtaining user info.</v-card-text>
          <v-card-actions>
            <v-spacer />
            <v-btn color="primary" variant="text" @click="retryLogin"> Retry</v-btn>
          </v-card-actions>
        </v-card>
        <v-card v-else>
          <v-card-title class="headline"> Obtaining user data</v-card-title>
          <v-progress-linear :indeterminate="true" class="mb-0" />
          <v-card-text class="pa-2"> Please wait until dialog close.</v-card-text>
        </v-card>
      </v-dialog>
    </v-row>
  </v-container>
</template>
