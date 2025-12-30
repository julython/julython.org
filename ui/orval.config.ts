import { defineConfig} from 'orval'

export default defineConfig({
  api: {
    input: 'http://localhost:8000/api/openapi.json',
    output: {
      mode: 'tags-split',
      target: 'src/api/endpoints.ts',
      client: 'react-query',
      baseUrl: '',
    },
  },
});