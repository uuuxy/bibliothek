// defineConfig kommt aus ESLint selbst; tseslint.config() ist seit typescript-eslint 8
// zugunsten dieser Variante als deprecated markiert (SonarQube javascript:S1874).
// tseslint bleibt für Parser und Regelsätze im Einsatz.
import { defineConfig } from 'eslint/config';
import eslintPluginSvelte from 'eslint-plugin-svelte';
import tseslint from 'typescript-eslint';
import globals from 'globals';
import eslintConfigPrettier from 'eslint-config-prettier';

export default defineConfig(
  ...tseslint.configs.recommended,
  ...eslintPluginSvelte.configs['flat/recommended'],
  {
    files: ['**/*.svelte', '**/*.js', '**/*.ts'],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser
      }
    },
    rules: {
      '@typescript-eslint/no-unused-vars': 'warn',
      '@typescript-eslint/ban-ts-comment': 'warn',
      '@typescript-eslint/no-explicit-any': 'warn'
    }
  },
  eslintConfigPrettier,
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node
      }
    }
  },
  {
    ignores: [
      "dist/",
      "build/",
      ".svelte-kit/",
      "node_modules/",
      "test-results/",
      "playwright-report/",
      "playwright.config.js"
    ]
  }
);
