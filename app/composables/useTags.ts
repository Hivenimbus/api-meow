import { ref, watch } from 'vue'
import { useApi, type ApiTag } from './useApi'

export interface Tag {
    id: string
    name: string
    color: string
}

const tags = ref<Tag[]>([])
const loading = ref(false)
const initialized = ref(false)

const availableColors = [
    '#10b981', // emerald
    '#3b82f6', // blue
    '#f59e0b', // amber
    '#8b5cf6', // violet
    '#ef4444', // red
    '#ec4899', // pink
    '#06b6d4', // cyan
    '#84cc16', // lime
]

export const useTags = () => {
    const api = useApi()

    const fetchTags = async () => {
        if (loading.value) return
        loading.value = true
        try {
            const data = await api.fetchTags()
            tags.value = data.map(t => ({
                id: t.id,
                name: t.name,
                color: t.color
            }))
            initialized.value = true
        } catch (e) {
            console.error('Failed to fetch tags:', e)
        } finally {
            loading.value = false
        }
    }

    const addTag = async (name: string, color: string) => {
        try {
            const newTag = await api.createTag(name, color)
            tags.value.push({
                id: newTag.id,
                name: newTag.name,
                color: newTag.color
            })
            return newTag
        } catch (e) {
            console.error('Failed to create tag:', e)
            throw e
        }
    }

    const updateTag = async (id: string, name: string, color: string) => {
        try {
            const updated = await api.updateTag(id, name, color)
            const tag = tags.value.find(t => t.id === id)
            if (tag) {
                tag.name = updated.name
                tag.color = updated.color
            }
        } catch (e) {
            console.error('Failed to update tag:', e)
            throw e
        }
    }

    const deleteTag = async (id: string) => {
        try {
            await api.deleteTag(id)
            const index = tags.value.findIndex(t => t.id === id)
            if (index > -1) {
                tags.value.splice(index, 1)
            }
        } catch (e) {
            console.error('Failed to delete tag:', e)
            throw e
        }
    }

    const getTagById = (id: string) => tags.value.find(t => t.id === id)

    // Initialize on first use
    if (!initialized.value && !loading.value) {
        fetchTags()
    }

    return {
        tags,
        loading,
        availableColors,
        fetchTags,
        addTag,
        updateTag,
        deleteTag,
        getTagById
    }
}
