<template>
  <div @click="$emit('settings')" class="group relative bg-slate-800/50 backdrop-blur-sm rounded-2xl border border-slate-700/50 p-5 hover:border-slate-600/50 hover:bg-slate-800/70 transition-all duration-300 hover:shadow-xl hover:shadow-emerald-500/5 cursor-pointer">
    <!-- Status Indicator Glow -->
    <div
      :class="[
        'absolute top-0 right-0 w-24 h-24 rounded-full blur-3xl opacity-20 transition-opacity duration-500',
        instance.status === 'connected' ? 'bg-emerald-500' : instance.status === 'connecting' ? 'bg-amber-500' : 'bg-slate-500'
      ]"
    />
    
    <!-- Header -->
    <div class="flex items-start justify-between mb-4">
      <div class="flex items-center gap-3">
        <div class="relative">
          <div class="w-12 h-12 rounded-xl bg-gradient-to-br from-emerald-500 to-teal-600 flex items-center justify-center shadow-lg shadow-emerald-500/20">
            <svg class="w-6 h-6 text-white" fill="currentColor" viewBox="0 0 24 24">
              <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/>
            </svg>
          </div>
          <!-- Status dot -->
          <div
            :class="[
              'absolute -bottom-0.5 -right-0.5 w-4 h-4 rounded-full border-2 border-slate-800',
              instance.status === 'connected' ? 'bg-emerald-500' : instance.status === 'connecting' ? 'bg-amber-500 animate-pulse' : 'bg-slate-500'
            ]"
          />
        </div>
        <div>
          <h3 class="text-white font-semibold text-lg">{{ instance.name }}</h3>
          <p class="text-slate-400 text-sm">{{ instance.phoneNumber || 'Não conectado' }}</p>
        </div>
      </div>
      
      <!-- Actions Menu -->
      <button
        @click.stop="$emit('delete')"
        class="relative z-10 p-2 rounded-lg text-slate-400 hover:text-red-400 hover:bg-red-500/10 transition-all opacity-0 group-hover:opacity-100"
        title="Excluir instância"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
        </svg>
      </button>
    </div>
    
    <!-- Badges Row -->
    <div class="flex flex-wrap items-center gap-2 mb-4">
      <!-- Status Badge -->
      <span
        :class="[
          'inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium',
          instance.status === 'connected' 
            ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20' 
            : instance.status === 'connecting' 
              ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
              : 'bg-slate-500/10 text-slate-400 border border-slate-500/20'
        ]"
      >
        <span
          :class="[
            'w-1.5 h-1.5 rounded-full',
            instance.status === 'connected' ? 'bg-emerald-400' : instance.status === 'connecting' ? 'bg-amber-400 animate-pulse' : 'bg-slate-400'
          ]"
        />
        {{ statusLabel }}
      </span>
      
      <!-- Tag Badge -->
      <span
        v-if="tag"
        class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium border"
        :style="{ backgroundColor: tag.color + '15', borderColor: tag.color + '30', color: tag.color }"
      >
        <span class="w-1.5 h-1.5 rounded-full" :style="{ backgroundColor: tag.color }" />
        {{ tag.name }}
      </span>
    </div>
    
    <!-- Connect Button -->
    <button
      @click="$emit('connect')"
      :disabled="instance.status === 'connected'"
      :class="[
        'w-full py-2.5 px-4 rounded-xl font-medium text-sm transition-all duration-300 flex items-center justify-center gap-2',
        instance.status === 'connected'
          ? 'bg-slate-700/50 text-slate-500 cursor-not-allowed'
          : 'bg-gradient-to-r from-emerald-500 to-teal-500 text-white hover:from-emerald-600 hover:to-teal-600 shadow-lg shadow-emerald-500/20 hover:shadow-emerald-500/30'
      ]"
    >
      <svg v-if="instance.status !== 'connected'" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
      </svg>
      <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
      </svg>
      {{ instance.status === 'connected' ? 'Conectado' : 'Conectar' }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useTags } from '~/composables/useTags'

export interface InstanceSettings {
  ignoreGroups: boolean
  webhookUrl: string
  webhookEvents: {
    receiveMessages: boolean
  }
}

export interface Instance {
  id: string
  name: string
  status: 'connected' | 'disconnected' | 'connecting'
  phoneNumber?: string
  tagId?: string
  settings: InstanceSettings
}

const props = defineProps<{
  instance: Instance
}>()

defineEmits<{
  connect: []
  delete: []
  settings: []
}>()

const { getTagById } = useTags()

const tag = computed(() => props.instance.tagId ? getTagById(props.instance.tagId) : null)

const statusLabel = computed(() => {
  const labels = {
    connected: 'Conectado',
    disconnected: 'Desconectado',
    connecting: 'Conectando...'
  }
  return labels[props.instance.status]
})
</script>

