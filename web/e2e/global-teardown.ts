import * as fs from 'node:fs'
import * as path from 'node:path'

const artifactRoot = process.env.PLAYWRIGHT_ARTIFACT_DIR ?? 'artifact'
const project = process.env.PLAYWRIGHT_PROJECT ?? 'desktop'

export default async function globalTeardown(): Promise<void> {
  const shots = path.join(artifactRoot, `screenshots-${project}`)
  const videos = path.join(artifactRoot, `videos-${project}`)
  fs.mkdirSync(shots, { recursive: true })
  fs.mkdirSync(videos, { recursive: true })

  for (const source of ['test-results', 'e2e/.shots']) {
    if (!fs.existsSync(source)) continue
    for (const entry of fs.readdirSync(source)) {
      const full = path.join(source, entry)
      if (fs.statSync(full).isFile() && entry.endsWith('.png')) {
        fs.copyFileSync(full, path.join(shots, entry))
        continue
      }
      if (!fs.statSync(full).isDirectory()) continue

      for (const png of fs.readdirSync(full).filter((f) => f.endsWith('.png')).sort()) {
        fs.copyFileSync(path.join(full, png), path.join(shots, `${entry}-${png}`))
      }
      const video = path.join(full, 'video.webm')
      if (fs.existsSync(video)) {
        fs.copyFileSync(video, path.join(videos, `${entry}.webm`))
      }
    }
  }
}
