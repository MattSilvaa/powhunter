import js from '@eslint/js'
import tseslint from '@typescript-eslint/eslint-plugin'
import tsparser from '@typescript-eslint/parser'
import react from 'eslint-plugin-react'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'

export default [
	{
		ignores: ['build/**', 'node_modules/**', 'dist/**', '.react-router/**'],
	},
	js.configs.recommended,
	{
		files: ['**/*.{ts,tsx}'],
		languageOptions: {
			parser: tsparser,
			parserOptions: {
				ecmaVersion: 'latest',
				sourceType: 'module',
				ecmaFeatures: {
					jsx: true,
				},
			},
			globals: {
				console: 'readonly',
				window: 'readonly',
				document: 'readonly',
				navigator: 'readonly',
				fetch: 'readonly',
				localStorage: 'readonly',
				sessionStorage: 'readonly',
				HTMLElement: 'readonly',
				HTMLInputElement: 'readonly',
				HTMLTextAreaElement: 'readonly',
				Event: 'readonly',
				Response: 'readonly',
				RequestCredentials: 'readonly',
				Deno: 'readonly',
			},
		},
		plugins: {
			'@typescript-eslint': tseslint,
			react,
			'react-hooks': reactHooks,
			'react-refresh': reactRefresh,
		},
		settings: {
			react: {
				version: 'detect',
			},
		},
		rules: {
			...tseslint.configs.recommended.rules,
			...react.configs.recommended.rules,
			...reactHooks.configs.recommended.rules,
			'react/react-in-jsx-scope': 'off',
			'react/prop-types': 'off',
			'@typescript-eslint/no-explicit-any': 'warn',
			'@typescript-eslint/no-unused-vars': [
				'error',
				{
					argsIgnorePattern: '^_',
					varsIgnorePattern: '^_',
				},
			],
			'react-refresh/only-export-components': [
				'warn',
				{ allowConstantExport: true },
			],
			'react/no-unescaped-entities': 'off',
		},
	},
	{
		files: ['**/*.test.{ts,tsx}', '**/*.spec.{ts,tsx}'],
		languageOptions: {
			globals: {
				global: 'readonly',
				setTimeout: 'readonly',
				clearTimeout: 'readonly',
				setInterval: 'readonly',
				clearInterval: 'readonly',
			},
		},
	},
]
