import { test, expect } from '@playwright/test'

const preExisting = process.env.PLAYWRIGHT_UPGRADE_CLIENT ?? 'upgrade-test'

test('a client issued before the upgrade is still listed', async ({ page }) => {
  await page.goto('/')

  await expect(page.getByTestId('clients-table')).toBeVisible({ timeout: 30_000 })
  await expect(page.getByTestId(`client-row-${preExisting}`)).toBeVisible({ timeout: 15_000 })
})

test('a client issued before the upgrade can still download its profile', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByTestId(`client-row-${preExisting}`)).toBeVisible({ timeout: 30_000 })

  const download = page.waitForEvent('download')
  await page.getByTestId(`client-download-${preExisting}`).click()
  const profile = await download

  expect(profile.suggestedFilename()).toBe(`${preExisting}.ovpn`)
})
