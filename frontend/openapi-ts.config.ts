import { defineConfig } from '@hey-api/openapi-ts'

export default defineConfig({
  input: '/openapi/openapi.yaml',
  output: {
    clean: true,
    path: 'src/api/generated',
    postProcess: ['prettier'],
  },
  plugins: [
    '@hey-api/typescript',
    {
      name: '@hey-api/client-fetch',
      runtimeConfigPath: './src/api/client',
    },
    '@hey-api/sdk',
  ],
})
