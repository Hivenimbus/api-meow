<template>
  <div class="min-h-screen p-6 sm:p-8 lg:p-12 w-full">
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

    <!-- Search and Filter Bar -->
    <div class="flex flex-col sm:flex-row gap-4 mb-6">
      <div class="relative flex-1 max-w-md">
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
      
      <!-- Tag Filter -->
      <div class="relative">
        <select
          v-model="selectedTagFilter"
          class="appearance-none px-4 py-3 pr-10 bg-slate-800/50 border border-slate-700/50 rounded-xl text-white focus:outline-none focus:border-emerald-500/50 focus:ring-2 focus:ring-emerald-500/20 transition-all cursor-pointer"
        >
          <option value="">Todas as etiquetas</option>
          <option v-for="tag in tags" :key="tag.id" :value="tag.id">{{ tag.name }}</option>
        </select>
        <svg class="absolute right-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400 pointer-events-none" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
        </svg>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="flex items-center justify-center py-16">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-500"></div>
    </div>

    <template v-else>
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
          @disconnect="handleDisconnect(instance)"
          @delete="openDeleteModal(instance)"
          @settings="openSettingsModal(instance)"
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
    </template>

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
        
        <div>
          <label class="block text-sm font-medium text-slate-300 mb-2">Webhook URL (opcional)</label>
          <input
            v-model="newInstance.webhookUrl"
            type="url"
            placeholder="https://seu-servidor.com/webhook"
            class="w-full px-4 py-3 bg-slate-900/50 border border-slate-700/50 rounded-xl text-white placeholder-slate-500 focus:outline-none focus:border-emerald-500/50 focus:ring-2 focus:ring-emerald-500/20 transition-all"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-slate-300 mb-2">Etiqueta (opcional)</label>
          <select
            v-model="newInstance.tagId"
            class="w-full px-4 py-3 bg-slate-900/50 border border-slate-700/50 rounded-xl text-white focus:outline-none focus:border-emerald-500/50 focus:ring-2 focus:ring-emerald-500/20 transition-all cursor-pointer"
          >
            <option value="">Sem etiqueta</option>
            <option v-for="tag in tags" :key="tag.id" :value="tag.id">{{ tag.name }}</option>
          </select>
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
            :disabled="!newInstance.name.trim() || saving"
            class="px-5 py-2 bg-gradient-to-r from-emerald-500 to-teal-500 text-white font-medium rounded-xl hover:from-emerald-600 hover:to-teal-600 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ saving ? 'Criando...' : 'Criar Instância' }}
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
            :disabled="saving"
            class="px-5 py-2 bg-red-500 text-white font-medium rounded-xl hover:bg-red-600 transition-all disabled:opacity-50"
          >
            {{ saving ? 'Excluindo...' : 'Excluir' }}
          </button>
        </div>
      </template>
    </Modal>

    <!-- Connect Modal (QR Code placeholder) -->
    <Modal v-model="showConnectModal" title="Conectar Instância" size="md">
      <div class="text-center py-4">
        <div class="w-64 h-64 mx-auto mb-4 bg-white rounded-2xl flex items-center justify-center overflow-hidden">
          <div v-if="qrCode" class="w-full h-full">
             <img :src="`data:image/png;base64,${qrCode}`" alt="QR Code" class="w-full h-full object-contain" />
          </div>
          <div v-else class="text-slate-800 text-center p-4">
            <template v-if="saving">
               <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-500 mx-auto mb-2"></div>
               <p class="text-sm font-medium">Gerando QR Code...</p>
            </template>
            <template v-else>
               <svg class="w-24 h-24 mx-auto mb-2 text-slate-600" fill="currentColor" viewBox="0 0 24 24">
                 <path d="M3 3h6v6H3V3zm2 2v2h2V5H5zm8-2h6v6h-6V3zm2 2v2h2V5h-2zM3 13h6v6H3v-6zm2 2v2h2v-2H5zm13-2h1v1h-1v-1zm-3 0h1v1h-1v-1zm-1 1h1v1h-1v-1zm2 0h1v1h-1v-1zm1 1h1v1h-1v-1zm-3 0h1v1h-1v-1zm4 0h1v1h-1v-1zm-1 1h1v1h-1v-1zm-3 0h1v1h-1v-1zm2 0h1v1h-1v-1zm1 1h1v1h-1v-1zm1 1h1v1h-1v-1zm-1 1h1v1h-1v-1zm1 0h1v1h-1v-1z"/>
               </svg>
               <p class="text-sm font-medium">QR Code</p>
            </template>
          </div>
        </div>
        <h4 class="text-lg font-semibold text-white mb-2">{{ instanceToConnect?.name }}</h4>
        <p class="text-slate-400 text-sm">
          Escaneie o QR Code com seu WhatsApp para conectar esta instância.
        </p>
      </div>
      <template #footer>
        <div class="flex justify-center gap-3">
           <button
            @click="closeConnectModal"
            class="px-4 py-2 text-slate-300 hover:text-white transition-colors"
          >
            Cancelar
          </button>
          <button
            v-if="!qrCode"
            @click="initiateConnection"
            :disabled="saving"
            class="px-5 py-2 bg-gradient-to-r from-emerald-500 to-teal-500 text-white font-medium rounded-xl hover:from-emerald-600 hover:to-teal-600 transition-all disabled:opacity-50"
          >
            {{ saving ? 'Gerando...' : 'Gerar QR Code' }}
          </button>
        </div>
      </template>
    </Modal>

    <!-- Instance Settings Modal -->
    <Modal v-model="showSettingsModal" :title="instanceToEdit?.name + ' - Configurações'" size="md">
      <div v-if="instanceToEdit" class="space-y-6">
        <!-- Ignore Groups -->
        <div class="flex items-center justify-between p-4 bg-slate-900/50 rounded-xl border border-slate-700/50">
          <div>
            <h4 class="text-white font-medium">Ignorar Grupos</h4>
            <p class="text-slate-400 text-sm">Não processar mensagens de grupos</p>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input 
              type="checkbox" 
              v-model="editSettings.ignoreGroups" 
              class="sr-only peer"
            />
            <div class="w-11 h-6 bg-slate-700 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-emerald-500/20 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-emerald-500"></div>
          </label>
        </div>

        <!-- Webhook URL -->
        <div>
          <label class="block text-sm font-medium text-slate-300 mb-2">Webhook URL</label>
          <input
            v-model="editSettings.webhookUrl"
            type="url"
            placeholder="https://seu-servidor.com/webhook"
            class="w-full px-4 py-3 bg-slate-900/50 border border-slate-700/50 rounded-xl text-white placeholder-slate-500 focus:outline-none focus:border-emerald-500/50 focus:ring-2 focus:ring-emerald-500/20 transition-all"
          />
        </div>

        <!-- Webhook Events -->
        <div>
          <label class="block text-sm font-medium text-slate-300 mb-3">Eventos do Webhook</label>
          <div class="space-y-2">
            <label class="flex items-center gap-3 p-3 bg-slate-900/50 rounded-xl border border-slate-700/50 cursor-pointer hover:border-slate-600/50 transition-all">
              <input 
                type="checkbox" 
                v-model="editSettings.receiveMessages"
                class="w-4 h-4 text-emerald-500 bg-slate-700 border-slate-600 rounded focus:ring-emerald-500/20 focus:ring-2"
              />
              <div>
                <span class="text-white font-medium">Receber Mensagens</span>
                <p class="text-slate-400 text-xs">Receber notificações quando novas mensagens chegarem</p>
              </div>
            </label>
          </div>
        </div>

        <!-- Tag -->
        <div>
          <label class="block text-sm font-medium text-slate-300 mb-2">Etiqueta</label>
          <select
            v-model="editSettings.tagId"
            class="w-full px-4 py-3 bg-slate-900/50 border border-slate-700/50 rounded-xl text-white focus:outline-none focus:border-emerald-500/50 focus:ring-2 focus:ring-emerald-500/20 transition-all cursor-pointer"
          >
            <option value="">Sem etiqueta</option>
            <option v-for="tag in tags" :key="tag.id" :value="tag.id">{{ tag.name }}</option>
          </select>
        </div>
      </div>
      <template #footer>
        <div class="flex gap-3 justify-end">
          <button
            @click="showSettingsModal = false"
            class="px-4 py-2 text-slate-300 hover:text-white transition-colors"
          >
            Fechar
          </button>
          <button
            @click="saveSettings"
            :disabled="saving"
            class="px-5 py-2 bg-gradient-to-r from-emerald-500 to-teal-500 text-white font-medium rounded-xl hover:from-emerald-600 hover:to-teal-600 transition-all disabled:opacity-50"
          >
            {{ saving ? 'Salvando...' : 'Salvar' }}
          </button>
        </div>
      </template>
    </Modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useToast } from '~/composables/useToast'
import { useTags } from '~/composables/useTags'
import { useApi } from '~/composables/useApi'
import type { Instance } from '~/components/InstanceCard.vue'

const { success, error } = useToast()
const { tags } = useTags()
const api = useApi()

// State
const searchQuery = ref('')
const selectedTagFilter = ref('')
const showAddModal = ref(false)
const showDeleteModal = ref(false)
const showConnectModal = ref(false)
const showSettingsModal = ref(false)
const instanceToDelete = ref<Instance | null>(null)
const instanceToConnect = ref<Instance | null>(null)
const instanceToEdit = ref<Instance | null>(null)
const loading = ref(true)
const saving = ref(false)

const newInstance = ref({
  name: '',
  tagId: '',
  webhookUrl: ''
})

const editSettings = ref({
  ignoreGroups: true,
  webhookUrl: '',
  receiveMessages: true,
  tagId: ''
})

// Instances from API
const instances = ref<Instance[]>([])

// Fetch instances on mount
onMounted(async () => {
  await fetchInstances()
})

const fetchInstances = async () => {
  loading.value = true
  try {
    const data = await api.fetchInstances()
    instances.value = data.map(i => ({
      id: i.id,
      name: i.name,
      status: i.status as 'connected' | 'disconnected' | 'connecting',
      phoneNumber: i.phoneNumber,
      tagId: i.tagId,
      settings: {
        ignoreGroups: i.ignoreGroups,
        webhookUrl: i.webhookUrl || '',
        webhookEvents: {
          receiveMessages: i.receiveMessages
        }
      }
    }))
  } catch (e) {
    error('Erro ao carregar instâncias')
    console.error(e)
  } finally {
    loading.value = false
  }
}

// Computed
const filteredInstances = computed(() => {
  let result = instances.value
  
  // Filter by tag
  if (selectedTagFilter.value) {
    result = result.filter(i => i.tagId === selectedTagFilter.value)
  }
  
  // Filter by search
  if (searchQuery.value.trim()) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(i => i.name.toLowerCase().includes(query))
  }
  
  return result
})

const connectedCount = computed(() => 
  instances.value.filter(i => i.status === 'connected').length
)

const disconnectedCount = computed(() => 
  instances.value.filter(i => i.status === 'disconnected').length
)

// Methods
const handleAddInstance = async () => {
  if (!newInstance.value.name.trim()) return
  
  saving.value = true
  try {
    const created = await api.createInstance({
      name: newInstance.value.name,
      webhookUrl: newInstance.value.webhookUrl || undefined,
      tagId: newInstance.value.tagId || undefined
    })
    
    instances.value.unshift({
      id: created.id,
      name: created.name,
      status: created.status as 'connected' | 'disconnected' | 'connecting',
      phoneNumber: created.phoneNumber,
      tagId: created.tagId,
      settings: {
        ignoreGroups: created.ignoreGroups,
        webhookUrl: created.webhookUrl || '',
        webhookEvents: {
          receiveMessages: created.receiveMessages
        }
      }
    })
    
    success(`Instância "${created.name}" criada com sucesso!`)
    
    newInstance.value.name = ''
    newInstance.value.tagId = ''
    newInstance.value.webhookUrl = ''
    showAddModal.value = false
  } catch (e) {
    error('Erro ao criar instância')
    console.error(e)
  } finally {
    saving.value = false
  }
}

const openDeleteModal = (instance: Instance) => {
  instanceToDelete.value = instance
  showDeleteModal.value = true
}

const handleDeleteInstance = async () => {
  if (!instanceToDelete.value) return
  
  saving.value = true
  try {
    await api.deleteInstance(instanceToDelete.value.name)
    const name = instanceToDelete.value.name
    instances.value = instances.value.filter(i => i.id !== instanceToDelete.value?.id)
    success(`Instância "${name}" excluída com sucesso!`)
    
    instanceToDelete.value = null
    showDeleteModal.value = false
  } catch (e) {
    error('Erro ao excluir instância')
    console.error(e)
  } finally {
    saving.value = false
  }
}

const handleConnect = (instance: Instance) => {
  instanceToConnect.value = instance
  showConnectModal.value = true
}

const openSettingsModal = (instance: Instance) => {
  instanceToEdit.value = instance
  editSettings.value = {
    ignoreGroups: instance.settings?.ignoreGroups ?? true,
    webhookUrl: instance.settings?.webhookUrl || '',
    receiveMessages: instance.settings?.webhookEvents?.receiveMessages ?? true,
    tagId: instance.tagId || ''
  }
  showSettingsModal.value = true
}

const saveSettings = async () => {
  if (!instanceToEdit.value) return
  
  saving.value = true
  try {
    const updated = await api.updateInstanceSettings(instanceToEdit.value.name, {
      webhookUrl: editSettings.value.webhookUrl || undefined,
      ignoreGroups: editSettings.value.ignoreGroups,
      receiveMessages: editSettings.value.receiveMessages,
      tagId: editSettings.value.tagId || undefined
    })
    
    // Update local instance
    const instance = instances.value.find(i => i.id === instanceToEdit.value?.id)
    if (instance) {
      instance.tagId = updated.tagId
      instance.settings = {
        ignoreGroups: updated.ignoreGroups,
        webhookUrl: updated.webhookUrl || '',
        webhookEvents: {
          receiveMessages: updated.receiveMessages
        }
      }
    }
    
    success(`Configurações da instância "${instanceToEdit.value.name}" salvas!`)
    showSettingsModal.value = false
  } catch (e) {
    error('Erro ao salvar configurações')
    console.error(e)
  } finally {
    saving.value = false
  }
}

const qrCode = ref('')
const connectionPollInterval = ref<NodeJS.Timeout | null>(null)

const initiateConnection = async () => {
  if (!instanceToConnect.value) return
  
  saving.value = true
  qrCode.value = ''
  
  try {
    const response = await api.connectInstance(instanceToConnect.value.name)
    
    // Update status
    const instance = instances.value.find(i => i.id === instanceToConnect.value?.id)
    if (instance) {
      instance.status = 'connecting'
    }

    if (response.qrCode) {
      qrCode.value = response.qrCode
    }
    
    // Start polling for status/QR updates
    startPolling(instanceToConnect.value.name)
    
  } catch (e) {
    error('Erro ao iniciar conexão')
    console.error(e)
    showConnectModal.value = false
  } finally {
    saving.value = false
  }
}

const startPolling = (instanceName: string) => {
  if (connectionPollInterval.value) clearInterval(connectionPollInterval.value)
  
  connectionPollInterval.value = setInterval(async () => {
    try {
      const statusData = await api.getWhatsAppStatus(instanceName)
      
      // Update instance status
      const instance = instances.value.find(i => i.name === instanceName)
      if (instance) {
        // Map backend status to frontend status if needed
        if (statusData.status === 'connected') {
          instance.status = 'connected'
          instance.phoneNumber = statusData.phone
          success(`Instância conectada: ${statusData.phone}`)
          stopPolling()
          showConnectModal.value = false
          return
        } else if (statusData.status === 'disconnected') {
           instance.status = 'disconnected'
        }
      }
      
      // If still connecting, try to get QR code if we don't have it or it expired
      if (statusData.status === 'connecting' || statusData.status === 'disconnected') {
         try {
            const qrData = await api.getQRCode(instanceName)
            if (qrData.qrCode) {
               qrCode.value = qrData.qrCode
            }
         } catch (e) {
            // Ignore error fetching QR code (might not be ready yet)
         }
      }

    } catch (e) {
      console.error('Error polling status:', e)
    }
  }, 2000)
}

const stopPolling = () => {
  if (connectionPollInterval.value) {
    clearInterval(connectionPollInterval.value)
    connectionPollInterval.value = null
  }
}

// Clean up polling when modal closes
const closeConnectModal = () => {
  stopPolling()
  showConnectModal.value = false
  qrCode.value = ''
}

const handleDisconnect = async (instance: Instance) => {
  if (!confirm(`Deseja desconectar a instância ${instance.name}?`)) return

  try {
    await api.disconnectInstance(instance.name)
    instance.status = 'disconnected'
    instance.phoneNumber = undefined
    success('Instância desconectada com sucesso')
  } catch (e) {
    error('Erro ao desconectar instância')
    console.error(e)
  }
}
</script>
