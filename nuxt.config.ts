// https://nuxt.com/docs/api/configuration/nuxt-config
const backendUrl = process.env.BACKEND_URL || 'http://localhost:8080'

export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  modules: ['@nuxtjs/tailwindcss'],
  runtimeConfig: {
    public: {
      backendUrl: process.env.BACKEND_URL || 'http://localhost:8080'
    }
  },
  nitro: {
    // Proxy all /api/* requests to the Go backend
    routeRules: {
      '/api/**': {
        proxy: `${backendUrl}/api/**`
      },
      '/instances/**': {
        proxy: `${backendUrl}/instances/**`
      },
      '/tags/**': {
        proxy: `${backendUrl}/tags/**`
      }
    }
  }
})