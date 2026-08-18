import { test, expect } from '@playwright/test'

test('clients page loads and a client can be created, downloaded and revoked', async ({ page }) => {
  await page.goto('/')

  await expect(page.getByTestId('clients-table')).toBeVisible({ timeout: 30_000 })

  await page.getByTestId('client-name').fill('smoke-client')
  await page.getByTestId('client-create').click()

  await expect(page.getByTestId('client-row-smoke-client')).toBeVisible({ timeout: 15_000 })

  const download = page.waitForEvent('download')
  await page.getByTestId('client-download-smoke-client').click()
  const profile = await download
  expect(profile.suggestedFilename()).toBe('smoke-client.ovpn')

  await page.getByTestId('client-revoke-smoke-client').click()
  await expect(page.getByTestId('revoke-dialog')).toBeVisible()
  await page.getByTestId('revoke-confirm').click()

  await expect(page.getByTestId('client-row-smoke-client')).toHaveCount(0, { timeout: 15_000 })
})

test('status page reports the server and its port', async ({ page }) => {
  await page.goto('/')
  await page.getByTestId('nav-status').click()

  await expect(page.getByTestId('status-address')).toBeVisible({ timeout: 30_000 })
  await expect(page.getByTestId('status-address')).toContainText(/:\d+\/(udp|tcp)/)
  await expect(page.getByTestId('connections-table')).toBeVisible()
})

test('settings page exposes the port for router forwarding', async ({ page }) => {
  await page.goto('/')
  await page.getByTestId('nav-settings').click()

  await expect(page.getByTestId('settings-port')).toBeVisible({ timeout: 30_000 })
  await expect(page.getByTestId('settings-save')).toBeEnabled()
})
