<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type Settings } from '@/api'
import { notify } from '@/notify'

const settings = ref<Settings | null>(null)
const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    settings.value = await api.settings()
  } catch (e) {
    notify.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!settings.value) return
  saving.value = true
  try {
    settings.value = await api.saveSettings(settings.value)
    notify.success('Saved, the VPN server is restarting')
  } catch (e) {
    notify.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div v-loading="loading" class="sc-card">
    <h1 class="sc-title">Settings</h1>
    <p class="sc-lead">
      Forward this port on your router to reach the VPN from outside your network. Changing it
      restarts the server, and existing client profiles must be downloaded again.
    </p>

    <el-form v-if="settings" label-position="top" class="settings-form">
      <el-form-item label="Port">
        <el-input-number
          v-model="settings.port"
          data-testid="settings-port"
          :min="1"
          :max="65535"
          controls-position="right"
        />
      </el-form-item>

      <el-form-item label="Protocol">
        <el-select v-model="settings.proto" data-testid="settings-proto" class="control">
          <el-option label="UDP" value="udp" />
          <el-option label="TCP" value="tcp" />
        </el-select>
      </el-form-item>

      <el-form-item label="Maximum clients">
        <el-input-number
          v-model="settings.max_clients"
          data-testid="settings-max-clients"
          :min="1"
          :max="255"
          controls-position="right"
        />
      </el-form-item>

      <el-form-item label="Data ciphers">
        <el-input
          v-model="settings.data_ciphers"
          data-testid="settings-data-ciphers"
          class="control"
        />
      </el-form-item>

      <el-form-item>
        <el-button type="primary" data-testid="settings-save" :loading="saving" @click="save">
          Save
        </el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<style scoped>
.settings-form {
  max-width: 420px;
}

.control {
  width: 100%;
}
</style>
