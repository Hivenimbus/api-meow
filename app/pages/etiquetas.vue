<template>
  <div class="min-h-screen p-6 sm:p-8 lg:p-12 w-full">
    <!-- Header -->
    <header class="mb-12">
      <div class="flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div>
          <h1 class="text-4xl font-bold text-white mb-3 tracking-tight">
            <span class="bg-gradient-to-r from-violet-400 to-purple-400 bg-clip-text text-transparent">
              Etiquetas
            </span>
          </h1>
          <p class="text-slate-400 text-lg">Organize suas instâncias com etiquetas personalizadas</p>
        </div>
        
        <button
          @click="openAddModal"
          class="inline-flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-violet-500 to-purple-500 text-white font-medium rounded-xl hover:from-violet-600 hover:to-purple-600 transition-all duration-300 shadow-lg shadow-violet-500/20 hover:shadow-violet-500/30 hover:-translate-y-0.5"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          Nova Etiqueta
        </button>
      </div>
    </header>

    <!-- Search Bar -->
    <div class="mb-8">
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
          placeholder="Buscar etiqueta..."
          class="w-full pl-12 pr-4 py-3 bg-slate-800/50 border border-slate-700/50 rounded-xl text-white placeholder-slate-400 focus:outline-none focus:border-violet-500/50 focus:ring-2 focus:ring-violet-500/20 transition-all"
        />
      </div>
    </div>

    <!-- Tags List -->
    <div v-if="filteredTags.length > 0" class="space-y-3">
      <div
        v-for="tag in filteredTags"
        :key="tag.id"
        class="group flex items-center justify-between p-4 bg-slate-800/50 backdrop-blur-sm rounded-xl border border-slate-700/50 hover:border-slate-600/50 hover:bg-slate-800/70 transition-all"
      >
        <div class="flex items-center gap-4">
          <div
            class="w-10 h-10 rounded-lg flex items-center justify-center"
            :style="{ backgroundColor: tag.color + '20' }"
          >
            <svg
              class="w-5 h-5"
              :style="{ color: tag.color }"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
            </svg>
          </div>
          <div>
            <h3 class="text-white font-medium">{{ tag.name }}</h3>
            <div class="flex items-center gap-2 mt-1">
              <span
                class="w-3 h-3 rounded-full"
                :style="{ backgroundColor: tag.color }"
              />
              <span class="text-xs text-slate-400">{{ tag.color }}</span>
            </div>
          </div>
        </div>
        
        <div class="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
          <button
            @click="openEditModal(tag)"
            class="p-2 rounded-lg text-slate-400 hover:text-violet-400 hover:bg-violet-500/10 transition-all"
            title="Editar"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
            </svg>
          </button>
          <button
            @click="openDeleteModal(tag)"
            class="p-2 rounded-lg text-slate-400 hover:text-red-400 hover:bg-red-500/10 transition-all"
            title="Excluir"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else class="flex flex-col items-center justify-center py-16">
      <div class="w-20 h-20 rounded-full bg-slate-800/50 flex items-center justify-center mb-4">
        <svg class="w-10 h-10 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
        </svg>
      </div>
      <h3 class="text-xl font-semibold text-white mb-2">
        {{ searchQuery ? 'Nenhuma etiqueta encontrada' : 'Nenhuma etiqueta criada' }}
      </h3>
      <p class="text-slate-400 text-center max-w-md">
        {{ searchQuery ? 'Tente buscar com outro termo.' : 'Crie etiquetas para organizar suas instâncias de WhatsApp.' }}
      </p>
      <button
        v-if="!searchQuery"
        @click="openAddModal"
        class="mt-6 inline-flex items-center gap-2 px-5 py-3 bg-gradient-to-r from-violet-500 to-purple-500 text-white font-medium rounded-xl hover:from-violet-600 hover:to-purple-600 transition-all"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        Criar Etiqueta
      </button>
    </div>

    <!-- Add/Edit Tag Modal -->
    <Modal v-model="showTagModal" :title="editingTag ? 'Editar Etiqueta' : 'Nova Etiqueta'" size="md">
      <form @submit.prevent="handleSaveTag" class="space-y-6">
        <div>
          <label class="block text-sm font-medium text-slate-300 mb-2">Nome da Etiqueta</label>
          <input
            v-model="tagForm.name"
            type="text"
            placeholder="Ex: VIP, Urgente..."
            class="w-full px-4 py-3 bg-slate-900/50 border border-slate-700/50 rounded-xl text-white placeholder-slate-500 focus:outline-none focus:border-violet-500/50 focus:ring-2 focus:ring-violet-500/20 transition-all"
            required
          />
        </div>
        
        <div>
          <label class="block text-sm font-medium text-slate-300 mb-3">Cor</label>
          <div class="flex flex-wrap gap-3">
            <button
              v-for="color in availableColors"
              :key="color"
              type="button"
              @click="tagForm.color = color"
              :class="[
                'w-10 h-10 rounded-xl transition-all duration-200',
                tagForm.color === color ? 'ring-2 ring-white ring-offset-2 ring-offset-slate-800 scale-110' : 'hover:scale-105'
              ]"
              :style="{ backgroundColor: color }"
            />
          </div>
        </div>
        
        <!-- Preview -->
        <div class="pt-4 border-t border-slate-700/50">
          <label class="block text-sm font-medium text-slate-300 mb-3">Prévia</label>
          <div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border" :style="{ backgroundColor: tagForm.color + '20', borderColor: tagForm.color + '40' }">
            <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: tagForm.color }" />
            <span class="text-sm font-medium" :style="{ color: tagForm.color }">{{ tagForm.name || 'Nome da etiqueta' }}</span>
          </div>
        </div>
      </form>
      <template #footer>
        <div class="flex gap-3 justify-end">
          <button
            @click="showTagModal = false"
            class="px-4 py-2 text-slate-300 hover:text-white transition-colors"
          >
            Cancelar
          </button>
          <button
            @click="handleSaveTag"
            :disabled="!tagForm.name.trim()"
            class="px-5 py-2 bg-gradient-to-r from-violet-500 to-purple-500 text-white font-medium rounded-xl hover:from-violet-600 hover:to-purple-600 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ editingTag ? 'Salvar' : 'Criar Etiqueta' }}
          </button>
        </div>
      </template>
    </Modal>

    <!-- Delete Confirmation Modal -->
    <Modal v-model="showDeleteModal" title="Excluir Etiqueta" size="sm">
      <div class="text-center">
        <div class="w-16 h-16 mx-auto mb-4 rounded-full bg-red-500/10 flex items-center justify-center">
          <svg class="w-8 h-8 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <h4 class="text-lg font-semibold text-white mb-2">Tem certeza?</h4>
        <p class="text-slate-400 text-sm">
          Você está prestes a excluir a etiqueta <strong class="text-white">{{ tagToDelete?.name }}</strong>. Esta ação não pode ser desfeita.
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
            @click="handleDeleteTag"
            class="px-5 py-2 bg-red-500 text-white font-medium rounded-xl hover:bg-red-600 transition-all"
          >
            Excluir
          </button>
        </div>
      </template>
    </Modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useTags, type Tag } from '~/composables/useTags'
import { useToast } from '~/composables/useToast'

const { tags, availableColors, addTag, updateTag, deleteTag, loading } = useTags()
const { success, error } = useToast()

const searchQuery = ref('')
const showTagModal = ref(false)
const showDeleteModal = ref(false)
const editingTag = ref<Tag | null>(null)
const tagToDelete = ref<Tag | null>(null)
const saving = ref(false)

const tagForm = ref({
  name: '',
  color: availableColors[0]
})

const filteredTags = computed(() => {
  if (!searchQuery.value.trim()) return tags.value
  const query = searchQuery.value.toLowerCase()
  return tags.value.filter(t => t.name.toLowerCase().includes(query))
})

const openAddModal = () => {
  editingTag.value = null
  tagForm.value = { name: '', color: availableColors[0] }
  showTagModal.value = true
}

const openEditModal = (tag: Tag) => {
  editingTag.value = tag
  tagForm.value = { name: tag.name, color: tag.color }
  showTagModal.value = true
}

const openDeleteModal = (tag: Tag) => {
  tagToDelete.value = tag
  showDeleteModal.value = true
}

const handleSaveTag = async () => {
  if (!tagForm.value.name.trim()) return
  
  saving.value = true
  try {
    if (editingTag.value) {
      await updateTag(editingTag.value.id, tagForm.value.name, tagForm.value.color)
      success(`Etiqueta "${tagForm.value.name}" atualizada!`)
    } else {
      await addTag(tagForm.value.name, tagForm.value.color)
      success(`Etiqueta "${tagForm.value.name}" criada!`)
    }
    showTagModal.value = false
  } catch (e) {
    error('Erro ao salvar etiqueta')
    console.error(e)
  } finally {
    saving.value = false
  }
}

const handleDeleteTag = async () => {
  if (!tagToDelete.value) return
  
  saving.value = true
  try {
    const name = tagToDelete.value.name
    await deleteTag(tagToDelete.value.id)
    success(`Etiqueta "${name}" excluída!`)
    
    tagToDelete.value = null
    showDeleteModal.value = false
  } catch (e) {
    error('Erro ao excluir etiqueta')
    console.error(e)
  } finally {
    saving.value = false
  }
}
</script>
