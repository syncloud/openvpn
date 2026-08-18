export type Client = {
  name: string
  serial: string
  created_at: string
  expires_at: string
  revoked: boolean
}

export type Connection = {
  name: string
  real_address: string
  virtual_ipv4: string
  virtual_ipv6: string
  bytes_received: number
  bytes_sent: number
  connected_at: string
  cipher: string
}

export type Status = {
  connections: Connection[]
  port: number
  proto: string
  server_address: string
  version: string
  running: boolean
}

export type Settings = {
  port: number
  proto: string
  data_ciphers: string
  auth: string
  keepalive: string
  max_clients: number
}

async function parse<T>(response: Response): Promise<T> {
  if (response.status === 401) {
    window.location.href = '/auth/login'
    throw new Error('unauthenticated')
  }
  if (!response.ok) {
    let message = `${response.status}`
    try {
      const body = await response.json()
      if (body?.error) message = body.error
    } catch {
      message = `${response.status}`
    }
    throw new Error(message)
  }
  return response.json() as Promise<T>
}

const jsonHeaders = { 'Content-Type': 'application/json' }

export const api = {
  listClients: () => fetch('/api/clients').then(parse<Client[]>),

  createClient: (name: string) =>
    fetch('/api/clients', {
      method: 'POST',
      headers: jsonHeaders,
      body: JSON.stringify({ name }),
    }).then(parse<{ name: string }>),

  revokeClient: (name: string) =>
    fetch(`/api/clients/${encodeURIComponent(name)}`, { method: 'DELETE' }).then(
      parse<{ name: string }>,
    ),

  configUrl: (name: string) => `/api/clients/${encodeURIComponent(name)}/config`,

  configText: async (name: string): Promise<string> => {
    const response = await fetch(api.configUrl(name))
    if (!response.ok) throw new Error(`${response.status}`)
    return response.text()
  },

  status: () => fetch('/api/status').then(parse<Status>),

  settings: () => fetch('/api/settings').then(parse<Settings>),

  saveSettings: (settings: Partial<Settings>) =>
    fetch('/api/settings', {
      method: 'POST',
      headers: jsonHeaders,
      body: JSON.stringify(settings),
    }).then(parse<Settings>),
}

export function formatBytes(bytes: number): string {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / Math.pow(1024, exponent)).toFixed(exponent === 0 ? 0 : 1)} ${units[exponent]}`
}
