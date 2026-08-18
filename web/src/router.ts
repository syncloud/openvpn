import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'clients', component: () => import('./views/Clients.vue') },
    { path: '/status', name: 'status', component: () => import('./views/Status.vue') },
    { path: '/settings', name: 'settings', component: () => import('./views/Settings.vue') },
  ],
})

export default router
