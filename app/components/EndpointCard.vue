<template>
  <div class="bg-slate-800/50 backdrop-blur-sm border border-slate-700/50 rounded-2xl overflow-hidden hover:border-slate-600/50 transition-all duration-300 group">
    <div class="p-6">
      <div class="flex items-start justify-between mb-4">
        <div>
          <h3 class="text-white font-semibold text-lg group-hover:text-emerald-400 transition-colors">{{ title }}</h3>
          <p class="text-slate-400 text-sm mt-1">{{ description }}</p>
        </div>
        <span :class="[
          'px-2 py-1 rounded text-xs font-bold font-mono border',
          methodClass
        ]">
          {{ method }}
        </span>
      </div>

      <div class="bg-slate-950/50 rounded-lg border border-slate-800/50 p-3 font-mono text-sm text-slate-300 mb-4 flex items-center gap-2">
        <span class="text-slate-500 select-none">$</span>
        <span class="text-emerald-400/80">{{ method }}</span>
        <span class="text-white">{{ path }}</span>
      </div>

      <div v-if="$slots.body" class="mt-4">
        <div class="text-xs text-slate-500 uppercase tracking-wider font-semibold mb-2">Request Body</div>
        <div class="bg-slate-950 rounded-xl border border-slate-800 p-4 font-mono text-xs text-slate-300 overflow-x-auto">
          <pre><slot name="body" /></pre>
        </div>
      </div>

      <div v-if="$slots.response" class="mt-4">
        <div class="text-xs text-emerald-500/80 uppercase tracking-wider font-semibold mb-2">Response Example</div>
        <div class="bg-slate-950 rounded-xl border border-emerald-500/20 p-4 font-mono text-xs text-emerald-400/90 overflow-x-auto">
          <pre><slot name="response" /></pre>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  method: string
  path: string
  title: string
  description: string
}>()

const methodClass = computed(() => {
  switch (props.method) {
    case 'GET': return 'bg-blue-500/10 text-blue-400 border-blue-500/20'
    case 'POST': return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
    case 'PUT': return 'bg-amber-500/10 text-amber-400 border-amber-500/20'
    case 'DELETE': return 'bg-red-500/10 text-red-400 border-red-500/20'
    default: return 'bg-slate-500/10 text-slate-400 border-slate-500/20'
  }
})
</script>
