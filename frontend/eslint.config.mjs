import js from "@eslint/js";
import globals from "globals";
import tseslint from "typescript-eslint";
import pluginVue from "eslint-plugin-vue";
import {defineConfig} from "eslint/config";

export default defineConfig([
    {
        ignores: ["node_modules/**/*", "dist/**/*", ".prettierrc.js", "src/vite-env.d.ts"],
    },
    {files: ["**/*.{js,mjs,cjs,ts,mts,cts,vue}"], plugins: {js}, extends: ["js/recommended"], languageOptions: {globals: globals.browser}},
    tseslint.configs.recommended,
    pluginVue.configs["flat/essential"],
    {files: ["**/*.vue"], languageOptions: {parserOptions: {parser: tseslint.parser}}},
    {
        rules: {
            "vue/multi-word-component-names": "off",
        }
    }
]);
