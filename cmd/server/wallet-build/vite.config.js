import { defineConfig } from 'vite';

export default defineConfig({
  build: {
    lib: {
      entry: './wallet.js',
      name: 'ReownWallet',
      fileName: 'wallet-reown-bundle',
      formats: ['umd']
    },
    outDir: '../static/js',
    emptyOutDir: false,
    rollupOptions: {
      output: {
        inlineDynamicImports: true
      }
    }
  }
});