<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { api, formatBytes, type Status } from '@/api'
import { notify } from '@/notify'

const status = ref<Status | null>(null)
const loading = ref(false)
let timer: ReturnType<typeof setInterval> | undefined

async function load() {
  loading.value = true
  try {
    status.value = await api.status()
  } catch (e) {
    notify.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  load()
  timer = setInterval(load, 10000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="sc-card">
    <h1 class="sc-title">Status</h1>

    <div v-if="status" class="server-info">
      <div class="sc-row">
        <span class="sc-row-label">Server</span>
        <span class="sc-row-value" data-testid="status-running">
          <el-tag :type="status.running ? 'success' : 'danger'" size="small">
            {{ status.running ? 'Running' : 'Stopped' }}
          </el-tag>
        </span>
      </div>
      <div class="sc-row">
        <span class="sc-row-label">Address</span>
        <span class="sc-row-value" data-testid="status-address">
          {{ status.server_address }}:{{ status.port }}/{{ status.proto }}
        </span>
      </div>
      <div v-if="status.version" class="sc-row">
        <span class="sc-row-label">Version</span>
        <span class="sc-row-value" data-testid="status-version">{{ status.version }}</span>
      </div>
    </div>

    <el-alert
      v-if="status && !status.running"
      type="warning"
      :closable="false"
      class="down-alert"
      title="The OpenVPN server is not responding on its management socket."
    />
  </div>

  <div class="sc-card">
    <h2 class="sc-section-title">Connected clients</h2>
    <el-table
      v-loading="loading"
      :data="status?.connections ?? []"
      data-testid="connections-table"
      empty-text="No clients connected"
    >
      <el-table-column prop="name" label="Name" min-width="120" />
      <el-table-column prop="virtual_ipv4" label="VPN IP" min-width="110" />
      <el-table-column prop="real_address" label="Source" min-width="150" />
      <el-table-column label="Received" min-width="100">
        <template #default="{ row }">{{ formatBytes(row.bytes_received) }}</template>
      </el-table-column>
      <el-table-column label="Sent" min-width="100">
        <template #default="{ row }">{{ formatBytes(row.bytes_sent) }}</template>
      </el-table-column>
      <el-table-column label="Since" min-width="150">
        <template #default="{ row }">{{ new Date(row.connected_at).toLocaleString() }}</template>
      </el-table-column>
    </el-table>
  </div>
</template>

<style scoped>
.server-info {
  margin-bottom: 8px;
}

.down-alert {
  margin-top: 16px;
}
</style>
