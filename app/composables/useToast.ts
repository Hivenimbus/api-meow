import { ref } from 'vue'

export interface Toast {
    id: number
    message: string
    type: 'success' | 'error' | 'warning' | 'info'
}

const toasts = ref<Toast[]>([])
let toastId = 0

export const useToast = () => {
    const addToast = (message: string, type: Toast['type'] = 'info', duration = 4000) => {
        const id = ++toastId
        toasts.value.push({ id, message, type })

        if (duration > 0) {
            setTimeout(() => {
                removeToast(id)
            }, duration)
        }
    }

    const removeToast = (id: number) => {
        const index = toasts.value.findIndex(t => t.id === id)
        if (index > -1) {
            toasts.value.splice(index, 1)
        }
    }

    const success = (message: string) => addToast(message, 'success')
    const error = (message: string) => addToast(message, 'error')
    const warning = (message: string) => addToast(message, 'warning')
    const info = (message: string) => addToast(message, 'info')

    return {
        toasts,
        addToast,
        removeToast,
        success,
        error,
        warning,
        info
    }
}
