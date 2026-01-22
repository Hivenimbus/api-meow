import { ref } from 'vue'

export interface Tag {
    id: string
    name: string
    color: string
}

const tags = ref<Tag[]>([
    { id: '1', name: 'Vendas', color: '#10b981' },
    { id: '2', name: 'Suporte', color: '#3b82f6' },
    { id: '3', name: 'Marketing', color: '#f59e0b' },
    { id: '4', name: 'Financeiro', color: '#8b5cf6' },
])

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
    const addTag = (name: string, color: string) => {
        const tag: Tag = {
            id: Date.now().toString(),
            name,
            color
        }
        tags.value.push(tag)
        return tag
    }

    const updateTag = (id: string, name: string, color: string) => {
        const tag = tags.value.find(t => t.id === id)
        if (tag) {
            tag.name = name
            tag.color = color
        }
    }

    const deleteTag = (id: string) => {
        const index = tags.value.findIndex(t => t.id === id)
        if (index > -1) {
            tags.value.splice(index, 1)
        }
    }

    const getTagById = (id: string) => tags.value.find(t => t.id === id)

    return {
        tags,
        availableColors,
        addTag,
        updateTag,
        deleteTag,
        getTagById
    }
}
