import { ExternalLink, FolderGit2, X } from 'lucide-preact'
import { useEffect, useMemo, useState } from 'preact/hooks'
import staticLicensesData from '../../data/licenses.json'
import {
	aboutModalInitialTab,
	config,
	displayVersion as formatVersion,
	isAboutModalOpen,
	mode,
} from '../../state/store'
import { copyToClipboard } from '../../utils/clipboard'
import { t, tf } from '../../utils/i18n'

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
	const displayVersion = formatVersion(build)

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

	async function copyLicenseText(text: string) {
		if (await copyToClipboard(text)) {
			setCopiedText(true)
			setTimeout(() => setCopiedText(false), 2000)
		}
	}

	if (!isOpen) return null

	return (
		<div
			class='modal modal-open z-50 bg-neutral/50 backdrop-blur-xs flex items-center justify-center p-2 sm:p-4'
			role='dialog'
			aria-modal='true'
			aria-label={t('arial_about_dialog')}
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
								{t('about_tagline')}
							</p>
						</div>
					</div>

					<button
						type='button'
						onClick={closeModal}
						class='btn btn-sm btn-circle btn-ghost text-base-content/60 hover:text-base-content'
						aria-label={t('arial_close_modal')}
						data-testid='about-modal-close'
					>
						<X class='size-4' />
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
						{t('about_tab_project')}
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
						<span>{t('about_tab_licenses')}</span>
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
									{t('about_overview_title')}
								</h3>
								<p class='text-xs/relaxed sm:text-sm/relaxed text-base-content/80'>
									{t('about_overview_text')}
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
										<FolderGit2 class='size-3.5' />
										{t('about_github_repo')}
									</a>
									<div class='badge badge-sm badge-ghost font-mono'>
										Go: {build?.go_version || '1.27'}
									</div>
									<div class='badge badge-sm badge-ghost font-mono'>
										{tf('about_platform', {
											platform: build?.platform || 'linux/amd64',
										})}
									</div>
								</div>
							</div>

							{/* Terminal Commands Guide */}
							<div class='rounded-2xl border border-base-200 bg-base-200/30 p-4 sm:p-5 flex flex-col gap-2'>
								<h3 class='font-bold text-sm text-base-content'>
									{t('about_cli_title')}
								</h3>
								<p class='text-xs text-base-content/70'>
									{t('about_cli_text')}
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
											{t('about_project_license')}
										</h3>
										<span class='badge badge-xs badge-success font-semibold'>
											{t('about_osi_approved')}
										</span>
									</div>
									<button
										type='button'
										onClick={() => copyLicenseText(report.project.text)}
										class='btn btn-xs btn-ghost text-xs'
									>
										{copiedText ? t('copied') : t('about_copy_license')}
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
										id='license-search'
										name='license-search'
										type='search'
										class='input input-sm w-full bg-base-200/60 focus:bg-base-100 text-xs'
										placeholder={t('about_license_search_placeholder')}
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
										{tf('about_filter_all', { count: report.packages.length })}
									</button>
									<button
										type='button'
										onClick={() => setEcosystemFilter('go')}
										data-testid='filter-go-btn'
										class={`btn btn-xs ${
											ecosystemFilter === 'go' ? 'btn-primary' : 'btn-ghost'
										}`}
									>
										{tf('about_filter_go', { count: goCount })}
									</button>
									<button
										type='button'
										onClick={() => setEcosystemFilter('npm')}
										data-testid='filter-npm-btn'
										class={`btn btn-xs ${
											ecosystemFilter === 'npm' ? 'btn-primary' : 'btn-ghost'
										}`}
									>
										{tf('about_filter_web', { count: npmCount })}
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
															title={t('arial_open_repository')}
															aria-label={tf('arial_open_repository_for', {
																name: pkg.name,
															})}
														>
															<span class='sr-only'>
																{tf('arial_open_repository_for', {
																	name: pkg.name,
																})}
															</span>
															<ExternalLink class='size-3.5' />
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
															{isExpanded
																? t('about_hide_license')
																: t('about_view_license')}
														</button>
													)}
												</div>
											</div>

											{/* Collapsible License Text */}
											{isExpanded && pkg.text && (
												<div class='pt-2 border-t border-base-200/80 animate-fadeIn'>
													<div class='flex items-center justify-between pb-1.5 text-xs text-base-content/60'>
														<span class='font-semibold text-[11px] uppercase tracking-wider'>
															{t('about_full_license_text')}
														</span>
														<button
															type='button'
															onClick={() => copyLicenseText(pkg.text || '')}
															class='link link-hover text-[11px]'
														>
															{copiedText ? t('copied') : t('copy')}
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
										{tf('about_no_dependencies_match', { query: searchQuery })}
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
