<template>
  <div class="fixed inset-0 min-h-screen w-full flex items-center justify-center p-6 z-[100] bg-slate-900">
    <div class="w-full max-w-md">
      <!-- Logo/Header -->
      <div class="text-center mb-8">
        <div class="w-20 h-20 mx-auto mb-4 rounded-2xl bg-gradient-to-br from-emerald-500 to-teal-500 flex items-center justify-center shadow-lg shadow-emerald-500/20">
          <svg class="w-10 h-10 text-white" fill="currentColor" viewBox="0 0 24 24">
            <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/>
          </svg>
        </div>
        <h1 class="text-3xl font-bold text-white mb-2">
          <span class="bg-gradient-to-r from-emerald-400 to-teal-400 bg-clip-text text-transparent">
            API Meow
          </span>
        </h1>
        <p class="text-slate-400">Insira sua API Key para acessar o sistema</p>
      </div>

      <!-- Login Form -->
      <div class="bg-slate-800/50 backdrop-blur-sm rounded-2xl border border-slate-700/50 p-8">
        <form @submit.prevent="handleLogin" class="space-y-6">
          <div>
            <label class="block text-sm font-medium text-slate-300 mb-2">API Key</label>
            <input
              v-model="apiKeyInput"
              type="password"
              placeholder="Insira sua API Key..."
              class="w-full px-4 py-3 bg-slate-900/50 border border-slate-700/50 rounded-xl text-white placeholder-slate-500 focus:outline-none focus:border-emerald-500/50 focus:ring-2 focus:ring-emerald-500/20 transition-all"
              :disabled="loading"
              required
            />
          </div>

          <div v-if="errorMessage" class="p-3 bg-red-500/10 border border-red-500/20 rounded-xl">
            <p class="text-red-400 text-sm text-center">{{ errorMessage }}</p>
          </div>

          <button
            type="submit"
            :disabled="loading || !apiKeyInput.trim()"
            class="w-full py-3 bg-gradient-to-r from-emerald-500 to-teal-500 text-white font-medium rounded-xl hover:from-emerald-600 hover:to-teal-600 transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
          >
            <svg v-if="loading" class="animate-spin h-5 w-5" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            {{ loading ? 'Verificando...' : 'Entrar' }}
          </button>
        </form>
      </div>

      <!-- Footer -->
      <p class="text-center text-slate-500 text-sm mt-6">
        A API Key é utilizada para autenticar suas requisições à API.
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useAuth } from '~/composables/useAuth'

definePageMeta({
  layout: false
})

const router = useRouter()
const { login, isAuthenticated, init } = useAuth()

const apiKeyInput = ref('')
const loading = ref(false)
const errorMessage = ref('')

// Check if already authenticated
onMounted(() => {
  init()
  if (isAuthenticated.value) {
    router.push('/dashboard')
  }
})

const handleLogin = async () => {
  if (!apiKeyInput.value.trim()) return

  loading.value = true
  errorMessage.value = ''

  try {
    const success = await login(apiKeyInput.value.trim())
    if (success) {
      router.push('/dashboard')
    } else {
      errorMessage.value = 'API Key inválida. Verifique e tente novamente.'
    }
  } catch (e) {
    errorMessage.value = 'Erro ao conectar. Verifique se o servidor está online.'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
/* Inherit dark background from app */
</style>
