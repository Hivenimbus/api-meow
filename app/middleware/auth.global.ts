// Middleware to protect routes - redirects to login if not authenticated
export default defineNuxtRouteMiddleware((to, from) => {
    // Skip middleware for login page
    if (to.path === '/login') {
        return
    }

    // Check authentication on client side
    if (import.meta.client) {
        const authKey = localStorage.getItem('api_meow_auth')
        if (!authKey) {
            return navigateTo('/login')
        }
    }
})
