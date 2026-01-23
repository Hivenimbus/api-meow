<template>
  <div class="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900">
    <!-- Show sidebar only when authenticated and not on login page -->
    <template v-if="showSidebar">
      <Sidebar ref="sidebarRef" />
      <main 
        :class="[
          'transition-all duration-300',
          sidebarRef?.isCollapsed ? 'ml-20' : 'ml-64'
        ]"
      >
        <NuxtPage />
      </main>
    </template>
    
    <!-- Login page - no sidebar -->
    <div v-else class="w-full">
      <NuxtPage />
    </div>
    
    <ToastContainer />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import Sidebar from '~/components/Sidebar.vue'
import { useAuth } from '~/composables/useAuth'

const route = useRoute()
const sidebarRef = ref<InstanceType<typeof Sidebar> | null>(null)
const { isAuthenticated, init } = useAuth()

// Initialize auth on mount
onMounted(() => {
  init()
})

// Show sidebar only when authenticated and not on login page
const showSidebar = computed(() => {
  return route.path !== '/login' && isAuthenticated.value
})
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

