// Auth composable - stores API key in localStorage
const AUTH_KEY = 'api_meow_auth'

export const useAuth = () => {
    const isAuthenticated = useState<boolean>('isAuthenticated', () => false)
    const apiKey = useState<string>('apiKey', () => '')

    // Initialize from localStorage on client side
    const init = () => {
        if (import.meta.client) {
            const stored = localStorage.getItem(AUTH_KEY)
            if (stored) {
                apiKey.value = stored
                isAuthenticated.value = true
            }
        }
    }

    const login = async (key: string): Promise<boolean> => {
        const config = useRuntimeConfig()
        const backendUrl = config.public.backendUrl

        try {
            // Validate API key by making a test request
            const response = await fetch(`${backendUrl}/api/tags`, {
                headers: {
                    'Authorization': `Bearer ${key}`
                }
            })

            if (response.ok) {
                apiKey.value = key
                isAuthenticated.value = true
                if (import.meta.client) {
                    localStorage.setItem(AUTH_KEY, key)
                }
                return true
            }
            return false
        } catch (e) {
            console.error('Login failed:', e)
            return false
        }
    }

    const logout = () => {
        apiKey.value = ''
        isAuthenticated.value = false
        if (import.meta.client) {
            localStorage.removeItem(AUTH_KEY)
        }
    }

    const getApiKey = (): string => {
        if (import.meta.client && !apiKey.value) {
            const stored = localStorage.getItem(AUTH_KEY)
            if (stored) {
                apiKey.value = stored
                isAuthenticated.value = true
            }
        }
        return apiKey.value
    }

    return {
        isAuthenticated,
        apiKey,
        init,
        login,
        logout,
        getApiKey
    }
}
