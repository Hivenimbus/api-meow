<template>
  <div class="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900">
    <Sidebar ref="sidebarRef" />
    <main 
      :class="[
        'transition-all duration-300',
        sidebarRef?.isCollapsed ? 'ml-20' : 'ml-64'
      ]"
    >
      <NuxtPage />
    </main>
    <ToastContainer />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Sidebar from '~/components/Sidebar.vue'

const sidebarRef = ref<InstanceType<typeof Sidebar> | null>(null)
</script>

<style>
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&display=swap');

* {
  font-family: 'Inter', sans-serif;
}

/* Custom scrollbar */
::-webkit-scrollbar {
  width: 8px;
}

::-webkit-scrollbar-track {
  background: rgba(15, 23, 42, 0.5);
}

::-webkit-scrollbar-thumb {
  background: rgba(100, 116, 139, 0.5);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: rgba(100, 116, 139, 0.7);
}
</style>
