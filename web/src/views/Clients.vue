<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { api, type Client } from '@/api'
import { notify } from '@/notify'

const clients = ref<Client[]>([])
const loading = ref(false)
const creating = ref(false)
const newName = ref('')
const revokeVisible = ref(false)
const revoking = ref(false)
const revokeTarget = ref<Client | null>(null)

const narrow = ref(false)

function updateWidth() {
  narrow.value = window.innerWidth < 768
}

const active = computed(() => clients.value.filter((c) => !c.revoked))
const revoked = computed(() => clients.value.filter((c) => c.revoked))

async function load() {
  loading.value = true
  try {
    clients.value = await api.listClients()
  } catch (e) {
    notify.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

async function create() {
  const name = newName.value.trim()
  if (!name) {
    notify.error('Name is required')
    return
  }
  creating.value = true
  try {
    await api.createClient(name)
    newName.value = ''
    notify.success(`Created ${name}`)
    await load()
  } catch (e) {
    notify.error((e as Error).message)
  } finally {
    creating.value = false
  }
}

function askRevoke(client: Client) {
  revokeTarget.value = client
  revokeVisible.value = true
}

async function confirmRevoke() {
  const client = revokeTarget.value
  if (!client) return
  revoking.value = true
  try {
    await api.revokeClient(client.name)
    notify.success(`Revoked ${client.name}`)
    revokeVisible.value = false
    revokeTarget.value = null
    await load()
  } catch (e) {
    notify.error((e as Error).message)
  } finally {
    revoking.value = false
  }
}

function download(client: Client) {
  window.location.href = api.configUrl(client.name)
}

onMounted(() => {
  updateWidth()
  window.addEventListener('resize', updateWidth)
  load()
})

onUnmounted(() => window.removeEventListener('resize', updateWidth))
</script>

<template>
  <div class="sc-card">
    <h1 class="sc-title">Clients</h1>
    <p class="sc-lead">
      Each client gets its own certificate. Download the profile and import it into the OpenVPN app
      on that device.
    </p>

    <div class="sc-actions create-row">
      <el-input
        v-model="newName"
        data-testid="client-name"
        placeholder="Device name, e.g. laptop"
        maxlength="64"
        class="create-input"
        @keyup.enter="create"
      />
      <el-button
        type="primary"
        data-testid="client-create"
        :loading="creating"
        @click="create"
      >
        Create
      </el-button>
    </div>

    <el-table
      v-loading="loading"
      :data="active"
      data-testid="clients-table"
      empty-text="No clients yet"
    >
      <el-table-column prop="name" label="Name" min-width="140">
        <template #default="{ row }">
          <span :data-testid="`client-row-${row.name}`">{{ row.name }}</span>
        </template>
      </el-table-column>
      <el-table-column v-if="!narrow" label="Expires" min-width="120">
        <template #default="{ row }">{{ new Date(row.expires_at).toLocaleDateString() }}</template>
      </el-table-column>
      <el-table-column label="Actions" :width="narrow ? 170 : 200" align="right">
        <template #default="{ row }">
          <div class="sc-actions row-actions">
            <el-button
              size="small"
              :data-testid="`client-download-${row.name}`"
              @click="download(row)"
            >
              Download
            </el-button>
            <el-button
              size="small"
              type="danger"
              plain
              :data-testid="`client-revoke-${row.name}`"
              @click="askRevoke(row)"
            >
              Revoke
            </el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>
  </div>

  <div v-if="revoked.length" class="sc-card">
    <h2 class="sc-section-title">Revoked</h2>
    <el-table :data="revoked" data-testid="revoked-table">
      <el-table-column prop="name" label="Name" min-width="140" />
      <el-table-column v-if="!narrow" prop="serial" label="Serial" min-width="180" />
    </el-table>
  </div>

  <el-dialog v-model="revokeVisible" title="Revoke client" width="420" data-testid="revoke-dialog">
    <p>
      Revoke <strong>{{ revokeTarget?.name }}</strong
      >? That device will no longer be able to connect, and the profile cannot be re-downloaded.
    </p>
    <template #footer>
      <el-button data-testid="revoke-cancel" @click="revokeVisible = false">Cancel</el-button>
      <el-button
        type="danger"
        data-testid="revoke-confirm"
        :loading="revoking"
        @click="confirmRevoke"
      >
        Revoke
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.create-row {
  margin-bottom: 18px;
}

.create-input {
  max-width: 320px;
}

.row-actions {
  justify-content: flex-end;
  gap: 6px;
}

</style>
