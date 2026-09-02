import { existsSync, mkdirSync } from 'node:fs'
import { resolve } from 'node:path'

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
		minify: true,
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

	console.log(
		`   ✓ Tailwind CSS compiled (${(css.length / 1024).toFixed(1)} KB)`,
	)
	console.log(`   ✓ Preact SPA bundled (${(js.length / 1024).toFixed(1)} KB)`)

	const template = await Bun.file(templatePath).text()

	// Replace stylesheet link with inlined <style> tag
	let output = template.replace(
		/<link\s+rel=["']stylesheet["']\s+href=["'][^"']+["']\s*\/?>/i,
		`<style>\n${css}\n</style>`,
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
		`<script>\n${js}\n</script>`,
	)

	if (!existsSync(distDir)) {
		mkdirSync(distDir, { recursive: true })
	}

	await Bun.write(distHtmlPath, output)

	const duration = (performance.now() - startTime).toFixed(0)
	const totalSize = (output.length / 1024).toFixed(1)

	console.log(
		`✅ Single-file bundle created: web/dist/index.html (${totalSize} KB in ${duration}ms)`,
	)
}

main().catch((err) => {
	console.error('❌ Bundle build failed:', err)
	process.exit(1)
})
