import { existsSync, mkdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { gzipSync } from 'node:zlib'

const startTime = performance.now()
const webDir = resolve(import.meta.dir, '..')
const templatePath = resolve(webDir, 'index.html')
const distDir = resolve(webDir, 'dist')
const distHtmlPath = resolve(distDir, 'index.html')

console.log('📦 Starting Jokateko single-file bundle build...')

// 1. Compile Tailwind CSS
async function buildCSS(): Promise<string> {
	const hasGlobalTailwind = Bun.which('tailwindcss') !== null
	const cmd = hasGlobalTailwind
		? ['tailwindcss', '-i', 'src/styles.css', '--minify']
		: ['bun', 'x', '@tailwindcss/cli', '-i', 'src/styles.css', '--minify']

	const proc = Bun.spawn(cmd, {
		cwd: webDir,
		stdout: 'pipe',
		stderr: 'pipe',
	})

	const [stdout, stderr] = await Promise.all([
		new Response(proc.stdout).text(),
		new Response(proc.stderr).text(),
	])

	const exitCode = await proc.exited
	if (exitCode !== 0) {
		throw new Error(
			`Tailwind CSS compilation failed (exit code ${exitCode}):\n${stderr}`,
		)
	}

	return stdout
}

// 2. Compile JavaScript (Preact, Signals, Valibot, Router, UI)
async function buildJS(): Promise<string> {
	const entrypoint = resolve(webDir, 'src/main.tsx')
	const result = await Bun.build({
		entrypoints: [entrypoint],
		minify: {
			whitespace: true,
			identifiers: true,
			syntax: true,
		},
		define: {
			'process.env.NODE_ENV': JSON.stringify('production'),
		},
		drop: ['debugger'],
		target: 'browser',
	})

	if (!result.success) {
		const errors = result.logs.map((log) => log.message).join('\n')
		throw new Error(`Bun.build failed:\n${errors}`)
	}

	if (result.outputs.length === 0) {
		throw new Error('Bun.build produced no output files')
	}

	return await result.outputs[0].text()
}

// 3. Assemble Single-File index.html
async function main() {
	if (!existsSync(templatePath)) {
		throw new Error(`HTML template not found at: ${templatePath}`)
	}

	const [css, js] = await Promise.all([buildCSS(), buildJS()])

	const { createHash } = await import('node:crypto')
	const scriptSha256 = createHash('sha256').update(js).digest('base64')
	const styleSha256 = createHash('sha256').update(css).digest('base64')

	const scriptHash = `sha256-${scriptSha256}`
	const styleHash = `sha256-${styleSha256}`

	console.log(
		`   ✓ Tailwind CSS compiled (${(css.length / 1024).toFixed(1)} KB) -> '${styleHash}'`,
	)
	console.log(
		`   ✓ Preact SPA bundled (${(js.length / 1024).toFixed(1)} KB) -> '${scriptHash}'`,
	)

	const template = await Bun.file(templatePath).text()

	// Replace stylesheet link with inlined <style> tag
	let output = template.replace(
		/<link\s+rel=["']stylesheet["']\s+href=["'][^"']+["']\s*\/?>/i,
		`<style>${css}</style>`,
	)

	// Ensure the payload placeholder is present inside jokateko-data script tag
	const placeholderScript = `<script id="jokateko-data" type="application/json">\n/* JOKATEKO_PAYLOAD_PLACEHOLDER */\n</script>`
	if (output.includes('id="jokateko-data"')) {
		output = output.replace(
			/<script\s+id=["']jokateko-data["'][^>]*>[\s\S]*?<\/script>/i,
			placeholderScript,
		)
	} else {
		// Fallback: inject right before app container
		output = output.replace(
			/<div\s+id=["']app["']/i,
			`${placeholderScript}\n  <div id="app"`,
		)
	}

	// Replace external script tag with inlined bundled JS
	output = output.replace(
		/<script\s+type=["']module["']\s+src=["'][^"']+["']\s*><\/script>/i,
		() => `<script>${js}</script>`,
	)

	if (!existsSync(distDir)) {
		mkdirSync(distDir, { recursive: true })
	}

	await Bun.write(distHtmlPath, output)
	const distGzPath = resolve(distDir, 'index.html.gz')
	const gzipped = gzipSync(Buffer.from(output, 'utf-8'), { level: 9 })
	await Bun.write(distGzPath, gzipped)

	await Bun.write(resolve(distDir, 'script.sha256'), scriptHash)
	await Bun.write(resolve(distDir, 'style.sha256'), styleHash)
	await Bun.write(
		resolve(distDir, 'hashes.json'),
		JSON.stringify(
			{
				script_hash: `'${scriptHash}'`,
				style_hash: `'${styleHash}'`,
				script_sha256: scriptSha256,
				style_sha256: styleSha256,
			},
			null,
			2,
		),
	)

	const duration = (performance.now() - startTime).toFixed(0)
	const rawSize = (output.length / 1024).toFixed(1)
	const gzSize = (gzipped.length / 1024).toFixed(1)

	console.log(
		`✅ Single-file bundle created: web/dist/index.html (${rawSize} KB, gzipped: ${gzSize} KB in ${duration}ms)`,
	)
}

main().catch((err) => {
	console.error('❌ Bundle build failed:', err)
	process.exit(1)
})
