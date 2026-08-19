import { chromium, type FullConfig } from '@playwright/test'
import { mkdir, writeFile } from 'node:fs/promises'
import { dirname } from 'node:path'
import { acceptConsent, credsFromEnv, loginOidc } from './helpers/syncloud'

const storageStatePath = 'e2e/.auth/user.json'

export default async function globalSetup(config: FullConfig): Promise<void> {
  const baseURL = config.projects[0].use.baseURL
  if (!baseURL) throw new Error('global-setup: baseURL not configured')

  await mkdir(dirname(storageStatePath), { recursive: true })

  const browser = await chromium.launch()
  const context = await browser.newContext({ ignoreHTTPSErrors: true, baseURL })
  const page = await context.newPage()

  try {
    const appOrigin = new URL(baseURL).origin
    const backAtApp = (url: URL) => url.origin === appOrigin && !url.pathname.startsWith('/auth/')

    await page.goto('/', { waitUntil: 'domcontentloaded' })
    await page.waitForURL(/^https:\/\/auth\./, { timeout: 30_000 })
    await mkdir('e2e/.shots', { recursive: true })
    await page.screenshot({ path: 'e2e/.shots/00-sso-login.png' })
    await loginOidc(page, credsFromEnv())
    await page.waitForURL((url) => url.pathname.includes('/consent/') || backAtApp(url), {
      timeout: 30_000,
    })
    if (page.url().includes('/consent/')) {
      await acceptConsent(page)
      await page.waitForURL(backAtApp, { timeout: 30_000 })
    }
    await context.storageState({ path: storageStatePath })
  } catch (err) {
    const body = await page.content()
    await mkdir('test-results', { recursive: true })
    await page.screenshot({ path: 'test-results/global-setup-fail.png', fullPage: true })
    await writeFile('test-results/global-setup-fail.html', body)
    throw err
  } finally {
    await browser.close()
  }
}
