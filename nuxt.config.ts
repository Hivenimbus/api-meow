// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  modules: ['@nuxtjs/tailwindcss'],
  runtimeConfig: {
    public: {
      // Use empty string to call same origin (via proxy) - works in both dev and prod
      backendUrl: ''
    }
  },
  nitro: {
    // Proxy all /api/* requests to the Go backend
    routeRules: {
      '/api/**': {
        proxy: 'http://localhost:8080/api/**'
      },
      '/instances/**': {
        proxy: 'http://localhost:8080/instances/**'
      },
      '/tags/**': {
        proxy: 'http://localhost:8080/tags/**'
      }
    }
  }
})