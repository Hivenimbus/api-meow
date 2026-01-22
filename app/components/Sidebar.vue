<template>
  <aside
    :class="[
      'fixed left-0 top-0 h-screen bg-slate-900/95 backdrop-blur-xl border-r border-slate-800/50 z-40 transition-all duration-300 flex flex-col',
      isCollapsed ? 'w-20' : 'w-64'
    ]"
  >
    <!-- Logo -->
    <div class="p-5 border-b border-slate-800/50">
      <div class="flex items-center gap-3">
        <div class="flex-shrink-0 w-10 h-10 rounded-xl bg-gradient-to-br from-emerald-500 to-teal-600 flex items-center justify-center shadow-lg shadow-emerald-500/20">
          <svg class="w-5 h-5 text-white" fill="currentColor" viewBox="0 0 24 24">
            <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/>
          </svg>
        </div>
        <div v-if="!isCollapsed" class="overflow-hidden">
          <h1 class="text-lg font-bold text-white truncate">API Meow</h1>
          <p class="text-xs text-slate-500">WhatsApp API</p>
        </div>
      </div>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 p-4 space-y-2">
      <NuxtLink
        to="/dashboard"
        :class="[
          'flex items-center gap-3 py-3 rounded-xl transition-all duration-200',
          isCollapsed ? 'justify-center px-0' : 'px-4',
          isActive('/dashboard')
            ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
            : 'text-slate-400 hover:text-white hover:bg-slate-800/50'
        ]"
      >
        <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
        </svg>
        <span v-if="!isCollapsed" class="font-medium">Dashboard</span>
      </NuxtLink>

      <NuxtLink
        to="/settings"
        :class="[
          'flex items-center gap-3 py-3 rounded-xl transition-all duration-200',
          isCollapsed ? 'justify-center px-0' : 'px-4',
          isActive('/settings')
            ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
            : 'text-slate-400 hover:text-white hover:bg-slate-800/50'
        ]"
      >
        <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
        <span v-if="!isCollapsed" class="font-medium">Configurações</span>
      </NuxtLink>
    </nav>


    <!-- Footer -->
    <div class="p-4 border-t border-slate-800/50 space-y-2">
      <a
        href="#"
        target="_blank"
        :class="[
          'flex items-center gap-3 py-3 rounded-xl transition-all duration-200',
          isCollapsed ? 'justify-center px-0' : 'px-4',
          'text-slate-400 hover:text-white hover:bg-slate-800/50'
        ]"
      >
        <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
        </svg>
        <span v-if="!isCollapsed" class="font-medium">Documentação</span>
      </a>

      <!-- Collapse Toggle -->
      <button
        @click="toggleCollapse"
        :class="[
          'w-full flex items-center gap-3 py-3 rounded-xl text-slate-500 hover:text-white hover:bg-slate-800/50 transition-all duration-200',
          isCollapsed ? 'justify-center px-0' : 'justify-center px-4'
        ]"
      >
        <svg
          :class="['w-5 h-5 transition-transform duration-300', isCollapsed ? 'rotate-180' : '']"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
        </svg>
      </button>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const route = useRoute()

const isCollapsed = ref(false)

const isActive = (path: string) => route.path === path || route.path.startsWith(path + '/')

const toggleCollapse = () => {
  isCollapsed.value = !isCollapsed.value
}

// Expose isCollapsed for parent to adjust main content margin
defineExpose({ isCollapsed })
</script>
