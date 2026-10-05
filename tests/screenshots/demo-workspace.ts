import { mkdirSync, rmSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { sampleSnapshot } from '../../web/src/fixtures/sampleData'

// Month the calendar screenshots show; open tasks get target dates inside it
export const DEMO_MONTH = '2026-10'

// JSON string escapes are valid TOML basic-string escapes
const str = (v: string) => JSON.stringify(v)
const list = (vs: string[] = []) => `[${vs.map(str).join(', ')}]`
// Sample ids like 'task-ui/ux-199' must match the server's ID pattern ^[a-z0-9][a-z0-9-]*$
const slug = (id: string) => id.toLowerCase().replace(/[^a-z0-9-]+/g, '-')

function entity(path: string, fields: Record<string, string | number | string[] | undefined>, body = '') {
	const lines = Object.entries(fields)
		.filter(([, v]) => v !== undefined && v !== '')
		.map(([k, v]) => `${k} = ${typeof v === 'number' ? v : Array.isArray(v) ? list(v) : str(v as string)}`)
	writeFileSync(path, `+++\n${lines.join('\n')}\n+++\n\n${body}\n`, 'utf8')
}

/**
 * Replaces the starter content of a freshly initialized workspace with the
 * Web UI sample snapshot, so screenshots show a realistic, busy board.
 */
export function seedDemoWorkspace(dir: string): void {
	const root = join(dir, '.jokateko')
	const { config, tasks, milestones, strategies, glossary } = sampleSnapshot

	writeFileSync(
		join(root, 'config.toml'),
		`version = "0"\n\n[project]\nname = ${str('Jokateko')}\ndescription = ${str(config.project.description ?? '')}\nlocale = "en-US"\n`,
		'utf8',
	)

	for (const kind of ['tasks', 'milestones', 'strategies', 'glossary']) {
		rmSync(join(root, kind), { recursive: true, force: true })
		mkdirSync(join(root, kind))
	}

	// Spread open tasks over working days of DEMO_MONTH so the calendar has content
	let open = 0
	for (const t of tasks) {
		let target_at: string | undefined
		if (['ready', 'in_progress', 'in_review'].includes(t.status) && open < 40) {
			const day = 1 + ((open * 7) % 31)
			const weekday = new Date(`${DEMO_MONTH}-${String(day).padStart(2, '0')}T00:00:00Z`).getUTCDay()
			if (weekday !== 0 && weekday !== 6) {
				target_at = `${DEMO_MONTH}-${String(day).padStart(2, '0')}T12:00:00Z`
			}
			open++
		}
		entity(
			join(root, 'tasks', `${slug(t.id)}.md`),
			{
				title: t.title,
				status: t.status,
				priority: t.priority,
				milestone: t.milestone ?? undefined,
				tags: t.tags,
				summary: t.summary,
				dependencies: t.dependencies?.length ? t.dependencies.map(slug) : undefined,
				target_at,
			},
			t.body ?? '',
		)
	}

	for (const m of milestones) {
		entity(
			join(root, 'milestones', `${m.id}.md`),
			{ title: m.title, status: m.status, target_date: m.target_date ?? undefined, tags: m.tags, summary: m.summary },
			m.body ?? '',
		)
	}

	for (const s of strategies) {
		entity(join(root, 'strategies', `${s.id}.md`), { title: s.title, tier: s.tier, tags: s.tags, summary: s.summary }, s.body ?? '')
	}

	for (const g of glossary) {
		entity(join(root, 'glossary', `${g.id}.md`), { title: g.title, tags: g.tags, summary: g.summary }, g.body ?? '')
	}
}
