import type { ChildProcess } from 'node:child_process'
import { spawn } from 'node:child_process'
import { existsSync, mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'

// The path does not depend on the folder that runs Playwright.
const binary = fileURLToPath(new URL('../bin/reviews-app', import.meta.url))

export interface RunningApp {
  baseURL: string
  stop: () => Promise<void>
}

// startApp starts the built app with a new data folder. stop deletes only
// this folder.
export async function startApp(port: number, env: Record<string, string> = {}): Promise<RunningApp> {
  if (!existsSync(binary)) {
    throw new Error(`${binary} does not exist. Run make build-app.`)
  }
  const dir = mkdtempSync(join(tmpdir(), 'reviews-e2e-'))
  const baseURL = `http://127.0.0.1:${port}`
  const child: ChildProcess = spawn(binary, ['serve', '--dir', dir, '--http', `127.0.0.1:${port}`], {
    env: { ...process.env, ...env },
    stdio: ['ignore', 'ignore', 'pipe'],
  })
  let stderr = ''
  child.stderr?.on('data', (chunk) => (stderr += chunk))

  async function stop() {
    if (child.exitCode === null) {
      const exited = new Promise((resolve) => child.once('exit', resolve))
      child.kill()
      await exited
    }
    rmSync(dir, { recursive: true, force: true })
  }

  for (let i = 0; i < 100; i++) {
    if (child.exitCode !== null) {
      await stop()
      throw new Error(`The app stopped with code ${child.exitCode}: ${stderr}`)
    }
    try {
      if ((await fetch(`${baseURL}/api/health`)).ok) {
        return { baseURL, stop }
      }
    } catch {
      // The app does not answer yet.
    }
    await new Promise((resolve) => setTimeout(resolve, 100))
  }
  await stop()
  throw new Error(`The app did not answer on ${baseURL}: ${stderr}`)
}
