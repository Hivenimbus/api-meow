import { useAuth } from './useAuth'

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
    const API_BASE_URL = `/api`

    // Instances
    const fetchInstances = async (): Promise<ApiInstance[]> => {
        return $fetch<ApiInstance[]>(`${API_BASE_URL}/instances`, {
            headers: headers()
        })
    }

    const createInstance = async (data: {
        name: string
        webhookUrl?: string
        tagId?: string
        ignoreGroups?: boolean
        receiveMessages?: boolean
    }): Promise<ApiInstance> => {
        return $fetch<ApiInstance>(`${API_BASE_URL}/instances`, {
            method: 'POST',
            headers: headers(),
            body: data
        })
    }

    const updateInstanceStatus = async (
        name: string,
        status: string,
        phoneNumber?: string
    ): Promise<ApiInstance> => {
        return $fetch<ApiInstance>(`${API_BASE_URL}/instances/${name}/status`, {
            method: 'PUT',
            headers: headers(),
            body: { status, phoneNumber }
        })
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
        return $fetch<ApiInstance>(`${API_BASE_URL}/instances/${name}/settings`, {
            method: 'PUT',
            headers: headers(),
            body: settings
        })
    }

    const deleteInstance = async (name: string): Promise<void> => {
        await $fetch(`${API_BASE_URL}/instances/${name}`, {
            method: 'DELETE',
            headers: headers()
        })
    }

    // Tags
    const fetchTags = async (): Promise<ApiTag[]> => {
        return $fetch<ApiTag[]>(`${API_BASE_URL}/tags`, {
            headers: headers()
        })
    }

    const createTag = async (name: string, color: string): Promise<ApiTag> => {
        return $fetch<ApiTag>(`${API_BASE_URL}/tags`, {
            method: 'POST',
            headers: headers(),
            body: { name, color }
        })
    }

    const updateTag = async (id: string, name: string, color: string): Promise<ApiTag> => {
        return $fetch<ApiTag>(`${API_BASE_URL}/tags/${id}`, {
            method: 'PUT',
            headers: headers(),
            body: { name, color }
        })
    }

    const deleteTag = async (id: string): Promise<void> => {
        await $fetch(`${API_BASE_URL}/tags/${id}`, {
            method: 'DELETE',
            headers: headers()
        })
    }

    // WhatsApp Connection
    const connectInstance = async (name: string): Promise<{
        status: string
        qrCode?: string
        phone?: string
        message: string
    }> => {
        try {
            return await $fetch<{ status: string; qrCode?: string; phone?: string; message: string }>(
                `${API_BASE_URL}/instances/${name}/connect`,
                { method: 'POST', headers: headers() }
            )
        } catch (err: any) {
            throw new Error(err?.data?.error || 'Failed to initiate connection')
        }
    }

    const getQRCode = async (name: string): Promise<{
        qrCode: string
        status: string
        phone?: string
    }> => {
        return $fetch<{ qrCode: string; status: string; phone?: string }>(
            `${API_BASE_URL}/instances/${name}/qrcode`,
            { headers: headers() }
        )
    }

    const getWhatsAppStatus = async (name: string): Promise<{
        status: string
        phone?: string
    }> => {
        return $fetch<{ status: string; phone?: string }>(
            `${API_BASE_URL}/instances/${name}/wa-status`,
            { headers: headers() }
        )
    }

    const testProxy = async (proxyUrl: string): Promise<{
        success: boolean
        ip?: string
        country?: string
        region?: string
        city?: string
        isp?: string
        org?: string
        waReachable?: boolean
        error?: string
    }> => {
        return $fetch<{
            success: boolean
            ip?: string
            country?: string
            region?: string
            city?: string
            isp?: string
            org?: string
            waReachable?: boolean
            error?: string
        }>(`${API_BASE_URL}/proxy/test`, {
            method: 'POST',
            headers: headers(),
            body: { proxyUrl }
        })
    }

    const disconnectInstance = async (name: string): Promise<void> => {
        await $fetch(`${API_BASE_URL}/instances/${name}/disconnect`, {
            method: 'POST',
            headers: headers()
        })
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
        disconnectInstance,
        // Proxy
        testProxy
    }
}
