// Plugins
import vue from "@vitejs/plugin-vue";
import { URL, fileURLToPath } from "node:url";
// Utilities
import { defineConfig } from "vite";
import vuetify, { transformAssetUrls } from "vite-plugin-vuetify";
import fs from "fs";
import path from "path";

let viteProxyRequestHeaders = {};
try {
  // Load the custom headers from the vite.local-http-headers.json file for local requests.
  viteProxyRequestHeaders = JSON.parse(fs.readFileSync(path.resolve("vite.local-http-headers.json"), "utf-8"));
} catch (e) {
  console.log("Did not load vite.env.json file.");
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue({
      template: { transformAssetUrls },
    }),
    // https://github.com/vuetifyjs/vuetify-loader/tree/master/packages/vite-plugin#readme
    vuetify({
      autoImport: true,
    }),
  ],
  define: { "process.env": {} },
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
    extensions: [".js", ".json", ".jsx", ".mjs", ".ts", ".tsx", ".vue"],
  },
  server: {
    port: 9051,
    proxy: {
      "/api": {
        target: "http://localhost:8081",
        changeOrigin: true,
        headers: viteProxyRequestHeaders
      },
    },
  },
});
