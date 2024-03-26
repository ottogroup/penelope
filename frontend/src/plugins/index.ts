import router from "@/router";
import { createPinia } from "pinia";
import { URL } from "url";
import { Environment } from "vitest";
import type { App } from "vue";

import registerConstants from "./constants";
import vuetify from "./vuetify";

if (import.meta.env.VITE_ENV === "local" && import.meta.env.VITE_LOG_LEVEL === "trace") {
  const { fetch: originalFetch } = window;
  window.fetch = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const response = await originalFetch(input, init);

    console.groupCollapsed("Starting Request", init?.method ?? "GET", input);
    console.time("Duration");
    try {
      const contentType = response.headers.get("Content-Type");
      if (contentType) {
        const jsonTypes = ["application/json", "application/problem+json"];
        const isJSON = jsonTypes.some((type) => contentType.toLowerCase().startsWith(type));

        let body;
        if (isJSON) {
          body = await response.clone().json();
        } else {
          body = await response.clone().text();
        }
        console.log("Response", response.statusText, `(${response.status})`, "Body:", JSON.stringify(body, null, 2));
      }
    } catch (error) {
      console.error(error);
    }
    console.timeEnd("Duration");
    console.groupEnd();

    return response;
  };
}

export function registerPlugins(app: App) {
  registerConstants(app);
  app.use(vuetify).use(router).use(createPinia());
}
