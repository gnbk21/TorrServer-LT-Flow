import js from '@eslint/js'
import ts from 'typescript-eslint'
import hooks from 'eslint-plugin-react-hooks'
export default ts.config(js.configs.recommended, ...ts.configs.recommended, {
  files: ['src/**/*.{ts,tsx}'],
  plugins: { 'react-hooks': hooks },
  languageOptions: { globals: { window: 'readonly', document: 'readonly', navigator: 'readonly', fetch: 'readonly', URL: 'readonly', URLSearchParams: 'readonly', FormData: 'readonly', Headers: 'readonly', AbortController: 'readonly', DOMException: 'readonly', console: 'readonly', setTimeout: 'readonly', clearTimeout: 'readonly', Blob: 'readonly', File: 'readonly' } },
  rules: { 'react-hooks/rules-of-hooks': 'error', 'react-hooks/exhaustive-deps': 'warn', '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }] },
})
