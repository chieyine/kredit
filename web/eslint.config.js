import js from '@eslint/js';
import ts from 'typescript-eslint';
import svelte from 'eslint-plugin-svelte';
import globals from 'globals';
import svelteConfig from './svelte.config.js';

/** Frontend lint rules.
 *
 * The Go side has run errcheck, govet, ineffassign, staticcheck and unused
 * since the start; the frontend had only a type check, so nothing caught
 * unused bindings, loose equality or stray console calls. These rules are
 * deliberately not type-aware: they run in seconds on every file including
 * the ones outside tsconfig, which is what makes them cheap enough to gate CI.
 */
export default ts.config(
	{
		ignores: [
			'.svelte-kit/**',
			'.vercel/**',
			'build/**',
			'node_modules/**',
			'test-results/**',
			'playwright-report/**',
			'src/lib/api/generated/**'
		]
	},
	js.configs.recommended,
	...ts.configs.recommended,
	...svelte.configs['flat/recommended'],
	{
		languageOptions: {
			globals: { ...globals.browser, ...globals.node }
		},
		rules: {
			// This app is served from the domain root and builds hrefs as plain
			// paths. resolve() exists for deployments under a base path, which
			// this is not, so the rule would cost 312 mechanical edits and buy
			// nothing. Revisit if a base path is ever introduced.
			'svelte/no-navigation-without-resolve': 'off',
			// Real, and worth fixing: an unkeyed each block reuses DOM nodes
			// across items, which shows the wrong row's state after a reorder.
			// There are ~200 of them and each needs a correct key chosen by
			// hand, so it lands as a warning rather than blocking CI today.
			'svelte/require-each-key': 'warn',
			// Svelte 5 migration advice, not a defect.
			'svelte/prefer-svelte-reactivity': 'warn',
			// A mustache holding a string with an escape sequence is not useless:
			// written as a plain attribute the \n would become a literal backslash-n.
			'svelte/no-useless-mustaches': ['error', { ignoreStringEscape: true }],
			'@typescript-eslint/no-unused-vars': [
				'error',
				{ argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' }
			],
			'@typescript-eslint/no-explicit-any': 'warn',
			eqeqeq: ['error', 'always', { null: 'ignore' }],
			'no-console': ['warn', { allow: ['warn', 'error'] }]
		}
	},
	{
		// Svelte single-file components carry TypeScript in <script lang="ts">,
		// so the script block needs the TypeScript parser explicitly.
		files: ['**/*.svelte', '**/*.svelte.ts', '**/*.svelte.js'],
		languageOptions: {
			parserOptions: { parser: ts.parser, svelteConfig }
		}
	},
	{
		files: ['**/*.svelte'],
		rules: {
			// `<\\/script>` inside a template literal is deliberate: it stops an
			// HTML parser ending the surrounding script early. Not a useless escape.
			'no-useless-escape': 'off'
		}
	},
	{
		files: ['src/service-worker.ts'],
		languageOptions: { globals: { ...globals.serviceworker } }
	},
	{
		files: ['tests/**', 'test-support/**', 'playwright.config.ts', '*.config.{js,ts}'],
		rules: { 'no-console': 'off', '@typescript-eslint/no-explicit-any': 'off' }
	}
);
