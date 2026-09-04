import { spawn } from 'node:child_process'
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

export interface TestServerInstance {
	dir: string
	url: string
	port: number
	stop: () => Promise<void>
}

export interface CustomTask {
	id: string
	title: string
	status: string
	priority: string
	tags: string[]
	summary: string
	body?: string
}

export interface CustomStrategy {
	id: string
	title: string
	tier: number
	summary: string
	tags?: string[]
	body?: string
}

export interface CustomGlossaryTerm {
	id: string
	title: string
	summary: string
	tags?: string[]
	body?: string
}

export async function startTestServer(options?: {
	customTasks?: CustomTask[]
	customStrategies?: CustomStrategy[]
	customGlossary?: CustomGlossaryTerm[]
}): Promise<TestServerInstance> {
	const dir = mkdtempSync(join(tmpdir(), 'jokateko-e2e-'))
	const binaryPath = '/workspaces/jokateko/bin/jokateko'

	// 1. Initialize workspace via jokateko init
	await new Promise<void>((resolve, reject) => {
		const initProc = spawn(binaryPath, ['init', '-dir', dir])
		initProc.on('exit', (code) => {
			if (code === 0) resolve()
			else reject(new Error(`jokateko init failed with code ${code}`))
		})
	})

	// 2. Write any custom tasks
	if (options?.customTasks) {
		for (const task of options.customTasks) {
			const taskPath = join(dir, '.jokateko', 'tasks', `${task.id}.md`)
			const content = `+++
id = "${task.id}"
title = "${task.title}"
status = "${task.status}"
priority = "${task.priority}"
tags = [${task.tags.map((t) => `"${t}"`).join(', ')}]
summary = "${task.summary}"
+++
${task.body || '## Acceptance Criteria\n- [ ] Task requirement\n'}
`
			writeFileSync(taskPath, content, 'utf8')
		}
	}

	// 2b. Write any custom strategies
	if (options?.customStrategies) {
		for (const s of options.customStrategies) {
			const stratPath = join(dir, '.jokateko', 'strategies', `${s.id}.md`)
			const content = `+++
title = "${s.title}"
tier = ${s.tier}
summary = "${s.summary}"
tags = [${(s.tags || []).map((t) => `"${t}"`).join(', ')}]
+++

${s.body || ''}
`
			writeFileSync(stratPath, content, 'utf8')
		}
	}

	// 2c. Write any custom glossary terms
	if (options?.customGlossary) {
		for (const g of options.customGlossary) {
			const glossPath = join(dir, '.jokateko', 'glossary', `${g.id}.md`)
			const content = `+++
title = "${g.title}"
summary = "${g.summary}"
tags = [${(g.tags || []).map((t) => `"${t}"`).join(', ')}]
+++

${g.body || ''}
`
			writeFileSync(glossPath, content, 'utf8')
		}
	}

	// 3. Launch daemon with port 0
	let srvProc: any
	let serverUrl = ''
	let serverPort = 0

	await new Promise<void>((resolve, reject) => {
		srvProc = spawn(binaryPath, ['serve', '-dir', dir, '-port', '0'])
		let output = ''

		srvProc.stdout?.on('data', (data: any) => {
			output += data.toString()
			const match = output.match(/http:\/\/127\.0\.0\.1:(\d+)/)
			if (match) {
				serverPort = Number.parseInt(match[1], 10)
				serverUrl = `http://127.0.0.1:${serverPort}`
				resolve()
			}
		})

		srvProc.stderr?.on('data', (data: any) => {
			output += data.toString()
		})

		srvProc.on('error', reject)
		srvProc.on('exit', (code: number) => {
			if (code !== 0 && !serverUrl) {
				reject(new Error(`Server exited prematurely with code ${code}: ${output}`))
			}
		})

		setTimeout(() => {
			if (!serverUrl) reject(new Error(`Timed out waiting for server to start: ${output}`))
		}, 10000)
	})

	// 4. Poll health endpoint until healthy
	let healthy = false
	for (let i = 0; i < 30; i++) {
		try {
			const res = await fetch(`${serverUrl}/api/health`)
			if (res.ok) {
				healthy = true
				break
			}
		} catch {
			// wait
		}
		await new Promise((r) => setTimeout(r, 100))
	}

	if (!healthy) {
		srvProc.kill('SIGKILL')
		rmSync(dir, { recursive: true, force: true })
		throw new Error(`Server at ${serverUrl} did not respond healthy`)
	}

	return {
		dir,
		url: serverUrl,
		port: serverPort,
		stop: async () => {
			if (srvProc && !srvProc.killed) {
				srvProc.kill('SIGTERM')
			}
			try {
				rmSync(dir, { recursive: true, force: true })
			} catch {
				// ignore cleanup error
			}
		},
	}
}
