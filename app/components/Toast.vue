<template>
  <Transition
    enter-active-class="transition-all duration-300 ease-out"
    enter-from-class="translate-x-full opacity-0"
    enter-to-class="translate-x-0 opacity-100"
    leave-active-class="transition-all duration-200 ease-in"
    leave-from-class="translate-x-0 opacity-100"
    leave-to-class="translate-x-full opacity-0"
  >
    <div
      v-if="visible"
      :class="[
        'fixed top-4 right-4 z-50 flex items-center gap-3 px-4 py-3 rounded-xl shadow-2xl backdrop-blur-sm border min-w-[300px] max-w-md',
        typeClasses
      ]"
    >
      <div :class="['flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center', iconBgClass]">
        <component :is="iconComponent" class="w-4 h-4" />
      </div>
      <div class="flex-1">
        <p class="text-sm font-medium text-white">{{ message }}</p>
      </div>
      <button
        @click="close"
        class="flex-shrink-0 text-white/60 hover:text-white transition-colors"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  visible: boolean
  message: string
  type: 'success' | 'error' | 'warning' | 'info'
}>()

const emit = defineEmits<{
  close: []
}>()

const typeClasses = computed(() => {
  const classes = {
    success: 'bg-emerald-500/20 border-emerald-500/30',
    error: 'bg-red-500/20 border-red-500/30',
    warning: 'bg-amber-500/20 border-amber-500/30',
    info: 'bg-blue-500/20 border-blue-500/30'
  }
  return classes[props.type]
})

const iconBgClass = computed(() => {
  const classes = {
    success: 'bg-emerald-500',
    error: 'bg-red-500',
    warning: 'bg-amber-500',
    info: 'bg-blue-500'
  }
  return classes[props.type]
})

const iconComponent = computed(() => {
  const icons = {
    success: 'IconCheck',
    error: 'IconX',
    warning: 'IconAlert',
    info: 'IconInfo'
  }
  return icons[props.type]
})

const close = () => emit('close')
</script>
