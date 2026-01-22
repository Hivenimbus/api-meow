<template>
  <div class="min-h-screen p-6 sm:p-8 lg:p-12 max-w-7xl mx-auto">
    <!-- Header -->
    <header class="mb-12">
      <div class="flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div>
          <h1 class="text-4xl font-bold text-white mb-3 tracking-tight">
            <span class="bg-gradient-to-r from-emerald-400 to-teal-400 bg-clip-text text-transparent">
              WhatsApp API
            </span>
          </h1>
          <p class="text-slate-400 text-lg">Gerencie suas instâncias do WhatsApp</p>
        </div>
        
        <button
          @click="showAddModal = true"
          class="inline-flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-emerald-500 to-teal-500 text-white font-medium rounded-xl hover:from-emerald-600 hover:to-teal-600 transition-all duration-300 shadow-lg shadow-emerald-500/20 hover:shadow-emerald-500/30 hover:-translate-y-0.5"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          Nova Instância
        </button>
      </div>
    </header>

    <!-- Search Bar -->
    <div class="mb-6">
      <div class="relative max-w-md">
        <svg
          class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Buscar instância..."
          class="w-full pl-12 pr-4 py-3 bg-slate-800/50 border border-slate-700/50 rounded-xl text-white placeholder-slate-400 focus:outline-none focus:border-emerald-500/50 focus:ring-2 focus:ring-emerald-500/20 transition-all"
        />
      </div>
    </div>

    <!-- Stats Cards -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
      <div class="bg-slate-800/30 backdrop-blur-sm rounded-xl border border-slate-700/30 p-4">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-lg bg-emerald-500/10 flex items-center justify-center">
            <svg class="w-5 h-5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-white">{{ connectedCount }}</p>
            <p class="text-sm text-slate-400">Conectadas</p>
          </div>
        </div>
      </div>
      <div class="bg-slate-800/30 backdrop-blur-sm rounded-xl border border-slate-700/30 p-4">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-lg bg-slate-500/10 flex items-center justify-center">
            <svg class="w-5 h-5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 5.636a9 9 0 010 12.728m0 0l-2.829-2.829m2.829 2.829L21 21M15.536 8.464a5 5 0 010 7.072m0 0l-2.829-2.829m-4.243 2.829a4.978 4.978 0 01-1.414-2.83m-1.414 5.658a9 9 0 01-2.167-9.238m7.824 2.167a1 1 0 111.414 1.414m-1.414-1.414L3 3m8.293 8.293l1.414 1.414" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-white">{{ disconnectedCount }}</p>
            <p class="text-sm text-slate-400">Desconectadas</p>
          </div>
        </div>
      </div>
      <div class="bg-slate-800/30 backdrop-blur-sm rounded-xl border border-slate-700/30 p-4">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-lg bg-blue-500/10 flex items-center justify-center">
            <svg class="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-white">{{ instances.length }}</p>
            <p class="text-sm text-slate-400">Total</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Instances Grid -->
    <div v-if="filteredInstances.length > 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      <InstanceCard
        v-for="instance in filteredInstances"
        :key="instance.id"
        :instance="instance"
        @connect="handleConnect(instance)"
        @delete="openDeleteModal(instance)"
      />
    </div>

    <!-- Empty State -->
    <div v-else class="flex flex-col items-center justify-center py-16">
      <div class="w-20 h-20 rounded-full bg-slate-800/50 flex items-center justify-center mb-4">
        <svg class="w-10 h-10 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
        </svg>
      </div>
      <h3 class="text-xl font-semibold text-white mb-2">
        {{ searchQuery ? 'Nenhuma instância encontrada' : 'Nenhuma instância criada' }}
      </h3>
      <p class="text-slate-400 text-center max-w-md">
        {{ searchQuery ? 'Tente buscar com outro termo.' : 'Crie sua primeira instância para começar a usar a API do WhatsApp.' }}
      </p>
      <button
        v-if="!searchQuery"
        @click="showAddModal = true"
        class="mt-6 inline-flex items-center gap-2 px-5 py-3 bg-gradient-to-r from-emerald-500 to-teal-500 text-white font-medium rounded-xl hover:from-emerald-600 hover:to-teal-600 transition-all"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        Criar Instância
      </button>
    </div>

    <!-- Add Instance Modal -->
    <Modal v-model="showAddModal" title="Nova Instância" size="md">
      <form @submit.prevent="handleAddInstance" class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-slate-300 mb-2">Nome da Instância</label>
          <input
            v-model="newInstance.name"
            type="text"
            placeholder="Ex: Vendas, Suporte..."
            class="w-full px-4 py-3 bg-slate-900/50 border border-slate-700/50 rounded-xl text-white placeholder-slate-500 focus:outline-none focus:border-emerald-500/50 focus:ring-2 focus:ring-emerald-500/20 transition-all"
            required
          />
        </div>
      </form>
      <template #footer>
        <div class="flex gap-3 justify-end">
          <button
            @click="showAddModal = false"
            class="px-4 py-2 text-slate-300 hover:text-white transition-colors"
          >
            Cancelar
          </button>
          <button
            @click="handleAddInstance"
            :disabled="!newInstance.name.trim()"
            class="px-5 py-2 bg-gradient-to-r from-emerald-500 to-teal-500 text-white font-medium rounded-xl hover:from-emerald-600 hover:to-teal-600 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Criar Instância
          </button>
        </div>
      </template>
    </Modal>

    <!-- Delete Confirmation Modal -->
    <Modal v-model="showDeleteModal" title="Excluir Instância" size="sm">
      <div class="text-center">
        <div class="w-16 h-16 mx-auto mb-4 rounded-full bg-red-500/10 flex items-center justify-center">
          <svg class="w-8 h-8 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <h4 class="text-lg font-semibold text-white mb-2">Tem certeza?</h4>
        <p class="text-slate-400 text-sm">
          Você está prestes a excluir a instância <strong class="text-white">{{ instanceToDelete?.name }}</strong>. Esta ação não pode ser desfeita.
        </p>
      </div>
      <template #footer>
        <div class="flex gap-3 justify-end">
          <button
            @click="showDeleteModal = false"
            class="px-4 py-2 text-slate-300 hover:text-white transition-colors"
          >
            Cancelar
          </button>
          <button
            @click="handleDeleteInstance"
            class="px-5 py-2 bg-red-500 text-white font-medium rounded-xl hover:bg-red-600 transition-all"
          >
            Excluir
          </button>
        </div>
      </template>
    </Modal>

    <!-- Connect Modal (QR Code placeholder) -->
    <Modal v-model="showConnectModal" title="Conectar Instância" size="md">
      <div class="text-center py-4">
        <div class="w-48 h-48 mx-auto mb-4 bg-white rounded-2xl flex items-center justify-center">
          <div class="text-slate-800 text-center p-4">
            <svg class="w-24 h-24 mx-auto mb-2 text-slate-600" fill="currentColor" viewBox="0 0 24 24">
              <path d="M3 3h6v6H3V3zm2 2v2h2V5H5zm8-2h6v6h-6V3zm2 2v2h2V5h-2zM3 13h6v6H3v-6zm2 2v2h2v-2H5zm13-2h1v1h-1v-1zm-3 0h1v1h-1v-1zm-1 1h1v1h-1v-1zm2 0h1v1h-1v-1zm1 1h1v1h-1v-1zm-3 0h1v1h-1v-1zm4 0h1v1h-1v-1zm-1 1h1v1h-1v-1zm-3 0h1v1h-1v-1zm2 0h1v1h-1v-1zm1 1h1v1h-1v-1zm1 1h1v1h-1v-1zm-1 1h1v1h-1v-1zm1 0h1v1h-1v-1z"/>
            </svg>
            <p class="text-sm font-medium">QR Code</p>
          </div>
        </div>
        <h4 class="text-lg font-semibold text-white mb-2">{{ instanceToConnect?.name }}</h4>
        <p class="text-slate-400 text-sm">
          Escaneie o QR Code com seu WhatsApp para conectar esta instância.
        </p>
      </div>
      <template #footer>
        <div class="flex justify-center">
          <button
            @click="simulateConnect"
            class="px-5 py-2 bg-gradient-to-r from-emerald-500 to-teal-500 text-white font-medium rounded-xl hover:from-emerald-600 hover:to-teal-600 transition-all"
          >
            Simular Conexão
          </button>
        </div>
      </template>
    </Modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useToast } from '~/composables/useToast'
import type { Instance } from '~/components/InstanceCard.vue'

const { success, error } = useToast()

// State
const searchQuery = ref('')
const showAddModal = ref(false)
const showDeleteModal = ref(false)
const showConnectModal = ref(false)
const instanceToDelete = ref<Instance | null>(null)
const instanceToConnect = ref<Instance | null>(null)

const newInstance = ref({
  name: ''
})

// Mock data
const instances = ref<Instance[]>([
  { id: '1', name: 'Vendas', status: 'connected', phoneNumber: '+55 11 99999-0001' },
  { id: '2', name: 'Suporte', status: 'disconnected' },
  { id: '3', name: 'Marketing', status: 'connecting' },
  { id: '4', name: 'Financeiro', status: 'connected', phoneNumber: '+55 11 99999-0004' },
])

// Computed
const filteredInstances = computed(() => {
  if (!searchQuery.value.trim()) return instances.value
  const query = searchQuery.value.toLowerCase()
  return instances.value.filter(i => i.name.toLowerCase().includes(query))
})

const connectedCount = computed(() => 
  instances.value.filter(i => i.status === 'connected').length
)

const disconnectedCount = computed(() => 
  instances.value.filter(i => i.status === 'disconnected').length
)

// Methods
const handleAddInstance = () => {
  if (!newInstance.value.name.trim()) return
  
  const instance: Instance = {
    id: Date.now().toString(),
    name: newInstance.value.name,
    status: 'disconnected'
  }
  
  instances.value.push(instance)
  success(`Instância "${instance.name}" criada com sucesso!`)
  
  newInstance.value.name = ''
  showAddModal.value = false
}

const openDeleteModal = (instance: Instance) => {
  instanceToDelete.value = instance
  showDeleteModal.value = true
}

const handleDeleteInstance = () => {
  if (!instanceToDelete.value) return
  
  const name = instanceToDelete.value.name
  instances.value = instances.value.filter(i => i.id !== instanceToDelete.value?.id)
  success(`Instância "${name}" excluída com sucesso!`)
  
  instanceToDelete.value = null
  showDeleteModal.value = false
}

const handleConnect = (instance: Instance) => {
  instanceToConnect.value = instance
  showConnectModal.value = true
}

const simulateConnect = () => {
  if (!instanceToConnect.value) return
  
  const instance = instances.value.find(i => i.id === instanceToConnect.value?.id)
  if (instance) {
    instance.status = 'connecting'
    showConnectModal.value = false
    
    // Simulate connection after 2 seconds
    setTimeout(() => {
      instance.status = 'connected'
      instance.phoneNumber = `+55 11 ${Math.floor(Math.random() * 90000000 + 10000000)}`
      success(`Instância "${instance.name}" conectada!`)
    }, 2000)
  }
}
</script>
