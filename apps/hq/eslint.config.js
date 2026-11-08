// Minimal ESLint until `bun lint` ships (#8): js + typescript-eslint + svelte,
// plus no-floating-promises (type-aware) on TS and Svelte.
import js from "@eslint/js";
import svelte from "eslint-plugin-svelte";
import globals from "globals";
import ts from "typescript-eslint";

export default ts.config(
  { ignores: [".svelte-kit/", "build/", "src/lib/contract.gen.ts"] },
  js.configs.recommended,
  ...ts.configs.recommended,
  ...svelte.configs.recommended,
  {
    languageOptions: {
      globals: { ...globals.browser, ...globals.node },
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
        extraFileExtensions: [".svelte"],
      },
    },
  },
  {
    files: ["**/*.svelte", "**/*.svelte.ts"],
    languageOptions: { parserOptions: { parser: ts.parser } },
  },
  {
    files: ["**/*.ts", "**/*.svelte"],
    rules: { "@typescript-eslint/no-floating-promises": "error" },
  },
  {
    files: ["**/*.js"],
    ...ts.configs.disableTypeChecked,
  },
);
