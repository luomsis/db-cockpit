import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173, host: true,
    /* 开发期代理到本地 apiserver（生产由 nginx 反代 /api/ → apiserver:8090） */
    proxy: {
      '/api': { target: 'http://localhost:8090', changeOrigin: true },
    },
  },
});
