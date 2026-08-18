<script setup lang="ts">
import { useRoute } from 'vue-router'

const route = useRoute()
</script>

<template>
  <div class="app-shell">
    <header class="app-header">
      <div class="brand">
        <span class="brand-dot" />
        <span class="brand-text">OpenVPN</span>
      </div>
      <nav class="app-nav">
        <router-link to="/" data-testid="nav-clients" :class="{ active: route.name === 'clients' }">
          Clients
        </router-link>
        <router-link
          to="/status"
          data-testid="nav-status"
          :class="{ active: route.name === 'status' }"
        >
          Status
        </router-link>
        <router-link
          to="/settings"
          data-testid="nav-settings"
          :class="{ active: route.name === 'settings' }"
        >
          Settings
        </router-link>
      </nav>
      <a href="/auth/logout" data-testid="logout" class="logout">Logout</a>
    </header>

    <main class="sc-page">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.app-shell {
  min-height: 100vh;
}

.app-header {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 14px 24px;
  background: var(--sc-header-bg);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid var(--sc-border-soft);
  position: sticky;
  top: 0;
  z-index: 10;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 700;
  font-size: 18px;
  color: var(--sc-ink);
}

.brand-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--sc-primary);
  box-shadow: 0 0 0 4px var(--sc-primary-soft);
}

.app-nav {
  display: flex;
  gap: 4px;
  margin-left: auto;
}

.app-nav a {
  padding: 7px 14px;
  border-radius: var(--sc-control-radius);
  text-decoration: none;
  font-weight: 600;
  font-size: 14px;
  color: var(--sc-muted);
}

.app-nav a:hover {
  background: var(--sc-option-hover);
  color: var(--sc-primary);
}

.app-nav a.active {
  background: var(--sc-primary-soft);
  color: var(--sc-primary);
}

.logout {
  text-decoration: none;
  font-weight: 600;
  font-size: 14px;
  color: var(--sc-muted);
}

.logout:hover {
  color: var(--sc-primary);
}

@media (max-width: 768px) {
  .app-header {
    padding: 10px 12px;
    gap: 10px;
    flex-wrap: wrap;
  }
  .app-nav {
    order: 3;
    width: 100%;
    margin-left: 0;
  }
  .logout {
    margin-left: auto;
  }
}
</style>
