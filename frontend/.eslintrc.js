module.exports = {
  root: true,
  env: {
    node: true,
    es2021: true,
  },
  parserOptions: {
    parser: "@typescript-eslint/parser",
  },
  extends: [
    "@vue/typescript",
    "@vue/eslint-config-typescript",
    "eslint:recommended",
    "plugin:vue/vue3-essential",
    "plugin:vue/vue3-strongly-recommended",
    "plugin:vue/vue3-recommended",
    "plugin:prettier/recommended",
  ],
  plugins: [],
  rules: {
    "vue/multi-word-component-names": "off",
    "no-unused-vars": [
      2,
      {
        args: "after-used",
        argsIgnorePattern: "^_",
      },
    ],
    "vue/block-order": [
      "error",
      {
        order: ["script", "template", "style"],
      },
    ],
  },
};
