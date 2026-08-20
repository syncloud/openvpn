import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'

const client = process.env.PLAYWRIGHT_UPGRADE_CLIENT ?? 'upgrade-test'

test('create a client that must survive the upgrade', async ({ page }, info) => {
  await page.goto('/')
  await expect(page.getByTestId('clients-table')).toBeVisible({ timeout: 30_000 })

  if (await page.getByTestId(`client-row-${client}`).count()) {
    await shoot(page, info, 'clients-before-upgrade')
    return
  }

  await page.getByTestId('client-name').fill(client)
  await page.getByTestId('client-create').click()

  await expect(page.getByTestId(`client-row-${client}`)).toBeVisible({ timeout: 15_000 })
  await shoot(page, info, 'clients-before-upgrade')
})
