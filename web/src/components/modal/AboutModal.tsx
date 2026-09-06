import { useEffect, useMemo, useState } from 'preact/hooks'
import staticLicensesData from '../../data/licenses.json'
import {
	aboutModalInitialTab,
	config,
	isAboutModalOpen,
	mode,
} from '../../state/store'

export function normalizeExternalURL(raw?: string): string {
	if (!raw) return ''
	let url = raw.trim()
	url = url
		.replace(/^git\+/, '')
		.replace(/^git:\/\//, 'https://')
		.replace(/\.git$/, '')
	if (url.startsWith('ssh://git@github.com/')) {
		return `https://github.com/${url.slice('ssh://git@github.com/'.length)}`
	}
	if (url.startsWith('git@github.com:')) {
		return `https://github.com/${url.slice('git@github.com:'.length)}`
	}
	if (url.startsWith('github:')) {
		return `https://github.com/${url.slice(7)}`
	}
	if (url.startsWith('http://')) {
		return `https://${url.slice(7)}`
	}
	if (url.startsWith('https://')) {
		return url
	}
	if (url.startsWith('github.com/')) {
		return `https://${url}`
	}
	const parts = url.split('/')
	if (
		parts.length === 2
		&& !parts[0].includes('.')
		&& !parts[0].includes(':')
	) {
		return `https://github.com/${url}`
	}
	return url.includes('.') ? `https://${url}` : url
}

interface PackageLicense {
	name: string
	version: string
	license: string
	ecosystem: 'go' | 'npm'
	url?: string
	text?: string
}

interface ProjectLicense {
	name: string
	version: string
	license: string
	url: string
	text: string
}

interface LicenseReport {
	project: ProjectLicense
	packages: PackageLicense[]
}

export function AboutModal() {
	const isOpen = isAboutModalOpen.value
	const isLive = mode.value === 'live'
	const build = config.value.build
	const version =
		build?.version && build.version !== 'dev' ? build.version : '1.0.0'
	const displayVersion = version.startsWith('v') ? version : `v${version}`

	const [activeTab, setActiveTab] = useState<'about' | 'licenses'>(
		aboutModalInitialTab.value || 'about',
	)

	useEffect(() => {
		if (isOpen) {
			setActiveTab(aboutModalInitialTab.value || 'about')
		}
	}, [isOpen])
	const [searchQuery, setSearchQuery] = useState('')
	const [ecosystemFilter, setEcosystemFilter] = useState<'all' | 'go' | 'npm'>(
		'all',
	)
	const [expandedPkg, setExpandedPkg] = useState<string | null>(null)
	const [copiedText, setCopiedText] = useState(false)
	const [report, setReport] = useState<LicenseReport>(
		staticLicensesData as LicenseReport,
	)

	// In live mode, fetch latest about and licenses data from the daemon
	useEffect(() => {
		if (!isLive || !isOpen) return

		let cancelled = false
		fetch('/api/licenses')
			.then((res) => (res.ok ? res.json() : null))
			.then((data: LicenseReport | null) => {
				if (!cancelled && data && data.packages) {
					setReport(data)
				}
			})
			.catch(() => {
				// Fallback to static bundled data
			})

		return () => {
			cancelled = true
		}
	}, [isLive, isOpen])

	function closeModal() {
		isAboutModalOpen.value = false
		setExpandedPkg(null)
	}

	// Close on Escape key
	useEffect(() => {
		function handleKeyDown(e: KeyboardEvent) {
			if (e.key === 'Escape' && isAboutModalOpen.value) {
				closeModal()
			}
		}

		window.addEventListener('keydown', handleKeyDown)
		return () => window.removeEventListener('keydown', handleKeyDown)
	}, [])

	const filteredPackages = useMemo(() => {
		const q = searchQuery.trim().toLowerCase()
		return report.packages.filter((pkg) => {
			if (ecosystemFilter !== 'all' && pkg.ecosystem !== ecosystemFilter) {
				return false
			}
			if (!q) return true
			const inName = pkg.name.toLowerCase().includes(q)
			const inLicense = pkg.license.toLowerCase().includes(q)
			const inUrl = (pkg.url || '').toLowerCase().includes(q)
			return inName || inLicense || inUrl
		})
	}, [report.packages, searchQuery, ecosystemFilter])

	const goCount = report.packages.filter((p) => p.ecosystem === 'go').length
	const npmCount = report.packages.filter((p) => p.ecosystem === 'npm').length

	function copyLicenseText(text: string) {
		navigator.clipboard.writeText(text)
		setCopiedText(true)
		setTimeout(() => setCopiedText(false), 2000)
	}

	if (!isOpen) return null

	return (
		<div
			class='modal modal-open z-50 bg-neutral/50 backdrop-blur-xs flex items-center justify-center p-2 sm:p-4'
			role='dialog'
			aria-modal='true'
			aria-label='About Jokateko and Open Source Licenses'
			data-testid='about-modal'
			onClick={(e) => {
				if (e.target === e.currentTarget) closeModal()
			}}
			onKeyDown={(e) => {
				if (e.key === 'Escape') closeModal()
			}}
		>
			<div class='modal-box w-full max-w-4xl max-h-[90vh] p-0 flex flex-col bg-base-100 rounded-2xl border border-base-200 shadow-2xl overflow-hidden'>
				{/* Modal Top Header */}
				<div class='flex items-center justify-between px-5 py-4 border-b border-base-200 bg-base-100 shrink-0'>
					<div class='flex items-center gap-2.5'>
						<div class='size-8 rounded-xl bg-primary/10 text-primary flex items-center justify-center font-bold font-mono text-sm border border-primary/20 shadow-2xs'>
							J
						</div>
						<div class='flex flex-col'>
							<div class='flex items-center gap-2'>
								<h2 class='text-base/tight font-bold text-base-content sm:text-lg'>
									Jokateko
								</h2>
								<span class='badge badge-xs badge-primary font-mono font-medium'>
									{displayVersion}
								</span>
								<span class='badge badge-xs badge-ghost font-mono'>MIT</span>
							</div>
							<p class='text-xs/tight text-base-content/60'>
								Local, Markdown-driven Kanban & Task Management
							</p>
						</div>
					</div>

					<button
						type='button'
						onClick={closeModal}
						class='btn btn-sm btn-circle btn-ghost text-base-content/60 hover:text-base-content'
						aria-label='Close modal'
						data-testid='about-modal-close'
					>
						✕
					</button>
				</div>

				{/* Navigation Tabs */}
				<div class='px-5 border-b border-base-200 bg-base-200/30 flex items-center gap-2 shrink-0'>
					<button
						type='button'
						onClick={() => setActiveTab('about')}
						data-testid='about-tab-btn'
						class={`py-3 px-3 border-b-2 text-xs sm:text-sm font-semibold transition-all cursor-pointer ${
							activeTab === 'about'
								? 'border-primary text-primary'
								: 'border-transparent text-base-content/60 hover:text-base-content'
						}`}
					>
						About Project
					</button>
					<button
						type='button'
						onClick={() => setActiveTab('licenses')}
						data-testid='licenses-tab-btn'
						class={`py-3 px-3 border-b-2 text-xs sm:text-sm font-semibold transition-all cursor-pointer flex items-center gap-1.5 ${
							activeTab === 'licenses'
								? 'border-primary text-primary'
								: 'border-transparent text-base-content/60 hover:text-base-content'
						}`}
					>
						<span>Open Source Licenses</span>
						<span class='badge badge-xs badge-ghost font-mono opacity-80'>
							{report.packages.length}
						</span>
					</button>
				</div>

				{/* Tab Body */}
				<div class='flex-1 overflow-y-auto p-5 space-y-4'>
					{activeTab === 'about' ? (
						<div class='flex flex-col gap-5'>
							{/* Overview Card */}
							<div class='rounded-2xl border border-base-200 bg-base-200/30 p-4 sm:p-5 flex flex-col gap-3'>
								<h3 class='font-bold text-sm text-base-content'>
									Zero-Dependency, Tasks-as-Code Architecture
								</h3>
								<p class='text-xs/relaxed sm:text-sm/relaxed text-base-content/80'>
									Jokateko operates on a "Spec-First" philosophy, where
									version-controlled Markdown files inside your repository are
									the single source of truth. It compiles down to a single
									auditable, zero-CGO binary with embedded Preact and Model
									Context Protocol (MCP) support.
								</p>
								<div class='flex flex-wrap items-center gap-2 pt-1'>
									<a
										href={normalizeExternalURL(
											report.project.url || 'https://github.com/RJuho/jokateko',
										)}
										target='_blank'
										rel='noopener noreferrer'
										class='btn btn-xs sm:btn-sm btn-outline gap-1.5 text-xs'
									>
										<svg
											class='size-3.5'
											fill='currentColor'
											viewBox='0 0 24 24'
											aria-hidden='true'
										>
											<title>GitHub</title>
											<path
												fill-rule='evenodd'
												clip-rule='evenodd'
												d='M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z'
											/>
										</svg>
										GitHub Repository
									</a>
									<div class='badge badge-sm badge-ghost font-mono'>
										Go: {build?.go_version || '1.27'}
									</div>
									<div class='badge badge-sm badge-ghost font-mono'>
										Platform: {build?.platform || 'linux/amd64'}
									</div>
								</div>
							</div>

							{/* Terminal Commands Guide */}
							<div class='rounded-2xl border border-base-200 bg-base-200/30 p-4 sm:p-5 flex flex-col gap-2'>
								<h3 class='font-bold text-sm text-base-content'>
									Terminal CLI Commands
								</h3>
								<p class='text-xs text-base-content/70'>
									You can also explore build metadata and licenses directly from
									your command line:
								</p>
								<div class='bg-base-300/60 rounded-xl p-3 font-mono text-xs text-base-content/90 flex flex-col gap-1'>
									<div>
										<span class='text-primary font-bold'>$</span> jokateko about
									</div>
									<div>
										<span class='text-primary font-bold'>$</span> jokateko
										licenses
									</div>
									<div>
										<span class='text-primary font-bold'>$</span> jokateko
										licenses --full
									</div>
								</div>
							</div>

							{/* Project MIT License Card */}
							<div class='rounded-2xl border border-base-200 bg-base-200/30 p-4 sm:p-5 flex flex-col gap-3'>
								<div class='flex items-center justify-between'>
									<div class='flex items-center gap-2'>
										<h3 class='font-bold text-sm text-base-content'>
											Project License (MIT)
										</h3>
										<span class='badge badge-xs badge-success font-semibold'>
											OSI Approved
										</span>
									</div>
									<button
										type='button'
										onClick={() => copyLicenseText(report.project.text)}
										class='btn btn-xs btn-ghost text-xs'
									>
										{copiedText ? 'Copied!' : 'Copy License'}
									</button>
								</div>
								<pre class='text-[11px] font-mono leading-relaxed bg-base-100 p-3 rounded-xl border border-base-200 overflow-x-auto whitespace-pre-wrap text-base-content/85'>
									{report.project.text}
								</pre>
							</div>
						</div>
					) : (
						<div class='flex flex-col gap-4'>
							{/* Filter and Search Bar */}
							<div class='flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-2.5'>
								{/* Search Input */}
								<div class='relative flex-1'>
									<input
										type='search'
										class='input input-sm w-full bg-base-200/60 focus:bg-base-100 text-xs'
										placeholder='Search package name, license type...'
										value={searchQuery}
										onInput={(e) =>
											setSearchQuery((e.target as HTMLInputElement).value)
										}
										data-testid='license-search-input'
									/>
								</div>

								{/* Ecosystem Filters */}
								<div class='flex items-center gap-1 shrink-0'>
									<button
										type='button'
										onClick={() => setEcosystemFilter('all')}
										data-testid='filter-all-btn'
										class={`btn btn-xs ${
											ecosystemFilter === 'all' ? 'btn-primary' : 'btn-ghost'
										}`}
									>
										All ({report.packages.length})
									</button>
									<button
										type='button'
										onClick={() => setEcosystemFilter('go')}
										data-testid='filter-go-btn'
										class={`btn btn-xs ${
											ecosystemFilter === 'go' ? 'btn-primary' : 'btn-ghost'
										}`}
									>
										Go ({goCount})
									</button>
									<button
										type='button'
										onClick={() => setEcosystemFilter('npm')}
										data-testid='filter-npm-btn'
										class={`btn btn-xs ${
											ecosystemFilter === 'npm' ? 'btn-primary' : 'btn-ghost'
										}`}
									>
										Web ({npmCount})
									</button>
								</div>
							</div>

							{/* Packages List */}
							<div class='flex flex-col gap-2'>
								{filteredPackages.map((pkg) => {
									const isExpanded = expandedPkg === pkg.name
									return (
										<div
											key={`${pkg.ecosystem}-${pkg.name}`}
											data-testid='license-package-item'
											class='rounded-xl border border-base-200 bg-base-100 hover:border-primary/30 transition-all p-3 flex flex-col gap-2 shadow-2xs'
										>
											<div class='flex items-center justify-between gap-2 flex-wrap'>
												<div class='flex items-center gap-2 min-w-0 flex-wrap'>
													<span class='font-bold text-xs sm:text-sm text-base-content truncate'>
														{pkg.name}
													</span>
													<span class='badge badge-xs badge-ghost font-mono'>
														{pkg.version}
													</span>
													<span
														class={`badge badge-xs uppercase font-bold text-[9px] ${
															pkg.ecosystem === 'go'
																? 'badge-info text-info-content'
																: 'badge-secondary text-secondary-content'
														}`}
													>
														{pkg.ecosystem}
													</span>
													<span class='badge badge-xs badge-outline font-mono text-[10px]'>
														{pkg.license}
													</span>
												</div>

												<div class='flex items-center gap-1.5 shrink-0'>
													{pkg.url && (
														<a
															href={normalizeExternalURL(pkg.url)}
															target='_blank'
															rel='noopener noreferrer'
															class='btn btn-xs btn-ghost btn-square text-base-content/60 hover:text-primary'
															title='Open repository'
															aria-label={`Open repository for ${pkg.name}`}
														>
															<span class='sr-only'>{`Open repository for ${pkg.name}`}</span>
															<svg
																class='size-3.5'
																fill='none'
																viewBox='0 0 24 24'
																stroke='currentColor'
																stroke-width='2'
																aria-hidden='true'
															>
																<title>Open repository</title>
																<path
																	stroke-linecap='round'
																	stroke-linejoin='round'
																	d='M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14'
																/>
															</svg>
														</a>
													)}
													{pkg.text && (
														<button
															type='button'
															onClick={() =>
																setExpandedPkg(isExpanded ? null : pkg.name)
															}
															data-testid='license-view-text-btn'
															class='btn btn-xs btn-ghost text-xs text-primary'
														>
															{isExpanded ? 'Hide License' : 'View License'}
														</button>
													)}
												</div>
											</div>

											{/* Collapsible License Text */}
											{isExpanded && pkg.text && (
												<div class='pt-2 border-t border-base-200/80 animate-fadeIn'>
													<div class='flex items-center justify-between pb-1.5 text-xs text-base-content/60'>
														<span class='font-semibold text-[11px] uppercase tracking-wider'>
															Full License Text
														</span>
														<button
															type='button'
															onClick={() => copyLicenseText(pkg.text || '')}
															class='link link-hover text-[11px]'
														>
															{copiedText ? 'Copied!' : 'Copy'}
														</button>
													</div>
													<pre class='text-[10px] font-mono leading-relaxed bg-base-200/50 p-3 rounded-lg border border-base-200 overflow-x-auto max-h-48 whitespace-pre-wrap text-base-content/85'>
														{pkg.text}
													</pre>
												</div>
											)}
										</div>
									)
								})}

								{filteredPackages.length === 0 && (
									<div class='py-12 text-center text-xs text-base-content/50 border border-dashed border-base-200 rounded-2xl'>
										No dependencies match "{searchQuery}"
									</div>
								)}
							</div>
						</div>
					)}
				</div>
			</div>
		</div>
	)
}
