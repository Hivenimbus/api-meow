import { useAuth } from './useAuth'

const getBackendUrl = () => {
    const config = useRuntimeConfig()
    return config.public.backendUrl || 'http://localhost:8080'
}

const headers = () => {
    const { getApiKey } = useAuth()
    const apiKey = getApiKey()
    return {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${apiKey}`
    }
}

export interface ApiInstance {
    id: string
    name: string
    status: string
    phoneNumber?: string
    tagId?: string
    ignoreGroups: boolean
    webhookUrl?: string
    receiveMessages: boolean
    proxyEnabled: boolean
    proxyUrl?: string
    createdAt: string
    updatedAt: string
}

export interface ApiTag {
    id: string
    name: string
    color: string
    createdAt: string
}

export const useApi = () => {
    const API_BASE_URL = `${getBackendUrl()}/api`

    // Instances
    const fetchInstances = async (): Promise<ApiInstance[]> => {
        const response = await fetch(`${API_BASE_URL}/instances`, {
            headers: headers()
        })
        if (!response.ok) throw new Error('Failed to fetch instances')
        return response.json()
    }

    const createInstance = async (data: {
        name: string
        webhookUrl?: string
        tagId?: string
        ignoreGroups?: boolean
        receiveMessages?: boolean
    }): Promise<ApiInstance> => {
        const response = await fetch(`${API_BASE_URL}/instances`, {
            method: 'POST',
            headers: headers(),
            body: JSON.stringify(data)
        })
        if (!response.ok) throw new Error('Failed to create instance')
        return response.json()
    }

    const updateInstanceStatus = async (
        name: string,
        status: string,
        phoneNumber?: string
    ): Promise<ApiInstance> => {
        const response = await fetch(`${API_BASE_URL}/instances/${name}/status`, {
            method: 'PUT',
            headers: headers(),
            body: JSON.stringify({ status, phoneNumber })
        })
        if (!response.ok) throw new Error('Failed to update instance status')
        return response.json()
    }

    const updateInstanceSettings = async (
        name: string,
        settings: {
            webhookUrl?: string
            ignoreGroups: boolean
            receiveMessages: boolean
            tagId?: string
            proxyEnabled: boolean
            proxyUrl?: string
        }
    ): Promise<ApiInstance> => {
        const response = await fetch(`${API_BASE_URL}/instances/${name}/settings`, {
            method: 'PUT',
            headers: headers(),
            body: JSON.stringify(settings)
        })
        if (!response.ok) throw new Error('Failed to update instance settings')
        return response.json()
    }

    const deleteInstance = async (name: string): Promise<void> => {
        const response = await fetch(`${API_BASE_URL}/instances/${name}`, {
            method: 'DELETE',
            headers: headers()
        })
        if (!response.ok) throw new Error('Failed to delete instance')
    }

    // Tags
    const fetchTags = async (): Promise<ApiTag[]> => {
        const response = await fetch(`${API_BASE_URL}/tags`, {
            headers: headers()
        })
        if (!response.ok) throw new Error('Failed to fetch tags')
        return response.json()
    }

    const createTag = async (name: string, color: string): Promise<ApiTag> => {
        const response = await fetch(`${API_BASE_URL}/tags`, {
            method: 'POST',
            headers: headers(),
            body: JSON.stringify({ name, color })
        })
        if (!response.ok) throw new Error('Failed to create tag')
        return response.json()
    }

    const updateTag = async (id: string, name: string, color: string): Promise<ApiTag> => {
        const response = await fetch(`${API_BASE_URL}/tags/${id}`, {
            method: 'PUT',
            headers: headers(),
            body: JSON.stringify({ name, color })
        })
        if (!response.ok) throw new Error('Failed to update tag')
        return response.json()
    }

    const deleteTag = async (id: string): Promise<void> => {
        const response = await fetch(`${API_BASE_URL}/tags/${id}`, {
            method: 'DELETE',
            headers: headers()
        })
        if (!response.ok) throw new Error('Failed to delete tag')
    }

    // WhatsApp Connection
    const connectInstance = async (name: string): Promise<{
        status: string
        qrCode?: string
        phone?: string
        message: string
    }> => {
        const response = await fetch(`${API_BASE_URL}/instances/${name}/connect`, {
            method: 'POST',
            headers: headers()
        })
        if (!response.ok) {
            const errorData = await response.json().catch(() => ({}))
            throw new Error(errorData.error || 'Failed to initiate connection')
        }
        return response.json()
    }

    const getQRCode = async (name: string): Promise<{
        qrCode: string
        status: string
        phone?: string
    }> => {
        const response = await fetch(`${API_BASE_URL}/instances/${name}/qrcode`, {
            headers: headers()
        })
        if (!response.ok) throw new Error('Failed to fetch QR code')
        return response.json()
    }

    const getWhatsAppStatus = async (name: string): Promise<{
        status: string
        phone?: string
    }> => {
        const response = await fetch(`${API_BASE_URL}/instances/${name}/wa-status`, {
            headers: headers()
        })
        if (!response.ok) throw new Error('Failed to fetch status')
        return response.json()
    }

    const disconnectInstance = async (name: string): Promise<void> => {
        const response = await fetch(`${API_BASE_URL}/instances/${name}/disconnect`, {
            method: 'POST',
            headers: headers()
        })
        if (!response.ok) throw new Error('Failed to disconnect')
    }

    return {
        // Instances
        fetchInstances,
        createInstance,
        updateInstanceStatus,
        updateInstanceSettings,
        deleteInstance,
        // Tags
        fetchTags,
        createTag,
        updateTag,
        deleteTag,
        // WhatsApp
        connectInstance,
        getQRCode,
        getWhatsAppStatus,
        disconnectInstance
    }
}
