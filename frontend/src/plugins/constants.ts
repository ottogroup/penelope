import type { App } from "vue";

export default function registerConstants(app: App) {
  app.config.globalProperties.$cardHeight = {
    xsm: 175,
    sm: 250,
    md: 315,
    lg: 550,
    xl: 650,
  };
  app.config.globalProperties.$maxURLLength = 1950;
}
