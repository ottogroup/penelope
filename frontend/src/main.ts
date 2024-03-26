/**
 * main.ts
 *
 * Bootstraps Vuetify and other plugins then mounts the App`
 */
import App from "@/App.vue";
import { registerPlugins } from "@/plugins";
import { createApp } from "vue";

const app = createApp(App);
registerPlugins(app);
app.mount("#app");
