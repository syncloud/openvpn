import type { Page, TestInfo } from '@playwright/test'

export async function shoot(page: Page, info: TestInfo, name: string): Promise<void> {
  await page.waitForTimeout(500)
  await page.screenshot({ path: info.outputPath(`${name}.png`), fullPage: false })
}
