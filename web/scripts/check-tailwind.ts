import { readdirSync, readFileSync, statSync, writeFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

const isFixMode = process.argv.includes('--fix')
const webDir = resolve(import.meta.dir, '..')

function getSourceFiles(dir: string): string[] {
	let files: string[] = []
	for (const item of readdirSync(dir)) {
		const full = join(dir, item)
		if (statSync(full).isDirectory()) {
			if (item !== 'node_modules' && item !== 'dist') {
				files = files.concat(getSourceFiles(full))
			}
		} else if (
			(item.endsWith('.tsx') || item.endsWith('.html'))
			&& !item.includes('test.')
		) {
			files.push(full)
		}
	}
	return files
}

interface ClassOccurrence {
	file: string
	line: number
	raw: string
}

interface Issue {
	file: string
	line: number
	raw: string
	canonical: string
	type: 'conflict' | 'non-canonical'
}

async function main() {
	const files = getSourceFiles(webDir)
	const occurrences: ClassOccurrence[] = []

	for (const file of files) {
		const content = readFileSync(file, 'utf8')
		const lines = content.split('\n')
		for (let i = 0; i < lines.length; i++) {
			const line = lines[i]
			const rx = /class(?:Name)?=(?:'([^']+)'|"([^"]+)")/g
			for (const match of line.matchAll(rx)) {
				const raw = (match[1] || match[2] || '').trim()
				if (raw && !raw.includes('${')) {
					occurrences.push({
						file,
						line: i + 1,
						raw,
					})
				}
			}
		}
	}

	const uniqueRawClasses = Array.from(new Set(occurrences.map((o) => o.raw)))
	const issues: Issue[] = []

	// Run canonicalize in batches of 40
	const batchSize = 40
	for (let i = 0; i < uniqueRawClasses.length; i += batchSize) {
		const batch = uniqueRawClasses.slice(i, i + batchSize)
		const hasGlobalTailwind = Bun.which('tailwindcss') !== null
		const cmd = hasGlobalTailwind
			? [
					'tailwindcss',
					'canonicalize',
					'--css',
					'src/styles.css',
					'--format',
					'json',
					...batch,
				]
			: [
					'bun',
					'x',
					'@tailwindcss/cli',
					'canonicalize',
					'--css',
					'src/styles.css',
					'--format',
					'json',
					...batch,
				]

		const proc = Bun.spawn(cmd, {
			cwd: webDir,
			stdout: 'pipe',
			stderr: 'pipe',
		})

		const text = await new Response(proc.stdout).text()
		await proc.exited

		const jsonStart = text.indexOf('[')
		if (jsonStart !== -1) {
			try {
				const parsed = JSON.parse(text.slice(jsonStart)) as Array<{
					input: string
					output: string
					changed: boolean
				}>
				for (const item of parsed) {
					const inTokens = item.input.split(/\s+/).filter(Boolean)
					const outTokens = item.output.split(/\s+/).filter(Boolean)

					// A conflict occurs when classes are dropped because they target the same property
					const isConflict =
						outTokens.length < inTokens.length && inTokens.length > 1
					// A non-canonical class occurs when an arbitrary value has an idiomatic shorthand
					const hasNonCanonical = inTokens.some(
						(t) =>
							t === 'h-[100dvh]'
							|| t === 'w-[100%]'
							|| t === 'h-[100%]'
							|| t === 'h-[100vh]',
					)

					if (isConflict || hasNonCanonical) {
						const matchingOccurrences = occurrences.filter(
							(o) => o.raw === item.input,
						)
						for (const occ of matchingOccurrences) {
							issues.push({
								file: occ.file,
								line: occ.line,
								raw: item.input,
								canonical: item.output,
								type: isConflict ? 'conflict' : 'non-canonical',
							})
						}
					}
				}
			} catch {
				// Ignore JSON parse error
			}
		}
	}

	if (issues.length === 0) {
		console.log(
			'✅ Tailwind CSS classes check passed: 0 conflicts or non-canonical classes found.',
		)
		process.exit(0)
	}

	console.error(`❌ Found ${issues.length} Tailwind CSS class issues:\n`)
	for (const issue of issues) {
		const relFile = issue.file.replace(`${webDir}/`, '')
		if (issue.type === 'conflict') {
			console.error(
				`  • ${relFile}:${issue.line} [cssConflict]\n    Classes: "${issue.raw}"\n    Suggested canonical: "${issue.canonical}"\n`,
			)
		} else {
			console.error(
				`  • ${relFile}:${issue.line} [suggestCanonicalClasses]\n    Classes: "${issue.raw}"\n    Suggested canonical: "${issue.canonical}"\n`,
			)
		}
	}

	if (isFixMode) {
		const byFile = new Map<string, Issue[]>()
		for (const issue of issues) {
			const list = byFile.get(issue.file) || []
			list.push(issue)
			byFile.set(issue.file, list)
		}

		for (const [file, fileIssues] of byFile.entries()) {
			let content = readFileSync(file, 'utf8')
			for (const iss of fileIssues) {
				content = content.replaceAll(iss.raw, iss.canonical)
			}
			writeFileSync(file, content, 'utf8')
		}
		console.log(
			`✨ Automatically fixed ${issues.length} Tailwind CSS issues in source files.`,
		)
		process.exit(0)
	}

	console.error(
		'Run "bun run check:tailwind --fix" to automatically apply fixes.',
	)
	process.exit(1)
}

main().catch((err) => {
	console.error('Error running Tailwind check:', err)
	process.exit(1)
})
