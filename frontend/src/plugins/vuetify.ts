import "@mdi/font/css/materialdesignicons.css";
import { type ThemeDefinition, createVuetify } from "vuetify";
import { md3 } from "vuetify/blueprints";
import { aliases, mdi } from "vuetify/iconsets/mdi";
import "vuetify/styles";

const penelopeLight: ThemeDefinition = {
  dark: false,
  colors: {
    primary: "#021B33",
    secondary: "#cccefd",
    accent: "#021B33",

    error: "#FF5252",
    info: "#336ad6",
    success: "#4CAF50",
    warning: "#FFC107",

    background: "#FFFFFF",
    surface: "#FFFFFF",

    "app-background": "#f7f7f7",
    "app-dt-background": "#FFFFFF",
    "app-bar": "#FFFFFF",
    "app-navigation": "#FFFFFF",
  },
};

// https://vuetifyjs.com/en/introduction/why-vuetify/#feature-guides
export default createVuetify({
  blueprint: md3,
  theme: {
    defaultTheme: "penelopeLight",
    themes: {
      penelopeLight,
    },
  },
  defaults: {
    VChip: {
      rounded: true,
    },
  },
  icons: {
    defaultSet: "mdi",
    aliases,
    sets: {
      mdi,
    },
  },
});
