import { useEffect } from 'preact/hooks'
import { activeColumnDetailId, config } from '../../state/store'

export function ColumnDetailModal() {
	const columnId = activeColumnDetailId.value
	const boardCfg = config.value.board
	const column = boardCfg.columns.find((c) => c.id === columnId)
	const mcpInstructions = config.value.mcp?.instructions

	function closeModal() {
		activeColumnDetailId.value = null
	}

	useEffect(() => {
		function handleKeyDown(e: KeyboardEvent) {
			if (e.key === 'Escape') {
				closeModal()
			}
		}
		window.addEventListener('keydown', handleKeyDown)
		return () => window.removeEventListener('keydown', handleKeyDown)
	}, [])

	if (!columnId || !column) {
		return null
	}

	const creatableStates = boardCfg.creatable_states ?? ['backlog']
	const isCreatable =
		creatableStates.length === 0 || creatableStates.includes(column.id)
	const defaultCreateState = boardCfg.default_create_state || 'backlog'
	const isDefaultCreate = column.id === defaultCreateState

	const editableStates = boardCfg.editable_states ?? ['backlog']
	const isEditable =
		editableStates.length === 0 || editableStates.includes(column.id)

	return (
		<div
			class='modal modal-open z-50 bg-neutral/40 backdrop-blur-xs flex items-end sm:items-center justify-center p-0 sm:p-4'
			role='dialog'
			aria-modal='true'
			aria-label={`Column details: ${column.name}`}
			data-testid='column-detail-modal'
			onClick={(e) => {
				if (e.target === e.currentTarget) {
					closeModal()
				}
			}}
			onKeyDown={(e) => {
				if (e.key === 'Escape') {
					closeModal()
				}
			}}
		>
			<div class='modal-box w-full max-w-lg sm:rounded-2xl rounded-b-none p-5 sm:p-6 overflow-y-auto bg-base-100 shadow-2xl flex flex-col gap-4 border border-base-300 transition-all duration-200'>
				{/* Top Bar: Column title pill, status slug, and close button */}
				<div class='flex items-center justify-between gap-3 pb-1 border-b border-base-200'>
					<div class='flex items-center gap-2.5 flex-wrap min-w-0'>
						<span
							class='badge badge-sm font-bold text-slate-900 border-0 shadow-2xs px-2.5 py-1 select-none'
							style={{ backgroundColor: column.color }}
							data-testid='column-modal-badge'
						>
							{column.name}
						</span>
						<span
							class='text-xs font-mono text-base-content/60'
							data-testid='column-modal-id'
						>
							status: {column.id}
						</span>
					</div>
					<button
						type='button'
						class='btn btn-sm btn-circle btn-ghost text-base-content/70 hover:text-base-content'
						onClick={closeModal}
						aria-label='Close modal'
						data-testid='column-modal-close-btn'
					>
						✕
					</button>
				</div>

				{/* Role & Handled By Section */}
				<div class='flex flex-col gap-1.5'>
					<h4 class='text-xs font-semibold uppercase tracking-wider text-base-content/70'>
						Assigned Role / Handled By
					</h4>
					{column.handled_by ? (
						<div
							class='flex items-start gap-3 p-3 rounded-xl bg-base-200/60 border border-base-300'
							data-testid='column-modal-handled-by'
						>
							<div class='size-8 rounded-lg bg-primary/10 text-primary flex items-center justify-center shrink-0 mt-0.5'>
								<svg
									class='size-4'
									fill='none'
									viewBox='0 0 24 24'
									stroke='currentColor'
									stroke-width='2'
									aria-hidden='true'
								>
									<path
										stroke-linecap='round'
										stroke-linejoin='round'
										d='M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z'
									/>
								</svg>
							</div>
							<div class='flex-1 min-w-0'>
								<div class='flex items-center gap-2'>
									<span class='font-bold text-sm text-base-content'>
										{column.handled_by}
									</span>
									<span class='badge badge-xs badge-primary font-medium'>
										Assigned
									</span>
								</div>
								<p class='text-xs text-base-content/70 mt-0.5'>
									Tasks in this column are designated for handling by{' '}
									<span class='font-semibold text-base-content'>
										{column.handled_by}
									</span>
									.
								</p>
							</div>
						</div>
					) : (
						<div
							class='p-3 rounded-xl bg-base-200/40 border border-base-200 text-xs text-base-content/70 flex items-center gap-2'
							data-testid='column-modal-handled-by'
						>
							<span class='badge badge-xs badge-ghost'>Unassigned</span>
							<span>Open to any team member or AI agent persona.</span>
						</div>
					)}
				</div>

				{/* Workflow Instructions & Prompt Section */}
				<div class='flex flex-col gap-1.5'>
					<div class='flex items-center justify-between'>
						<h4 class='text-xs font-semibold uppercase tracking-wider text-base-content/70'>
							Workflow Guidance & Instructions
						</h4>
					</div>
					{column.instructions ? (
						<div
							class='rounded-xl border border-base-300 bg-base-200 p-3.5 font-sans text-sm/relaxed whitespace-pre-wrap text-base-content shadow-2xs'
							data-testid='column-modal-instructions'
						>
							{column.instructions}
						</div>
					) : (
						<div
							class='p-3.5 rounded-xl bg-base-200/40 border border-base-200 text-xs text-base-content/60 italic'
							data-testid='column-modal-instructions'
						>
							No column-specific workflow instructions configured. You can set{' '}
							<code class='font-mono text-xs bg-base-300 px-1 py-0.5 rounded'>
								instructions = "..."
							</code>{' '}
							under{' '}
							<code class='font-mono text-xs bg-base-300 px-1 py-0.5 rounded'>
								[[board.columns]]
							</code>{' '}
							in config.
						</div>
					)}
				</div>

				{/* Board Column Policies Grid */}
				<div class='flex flex-col gap-1.5'>
					<h4 class='text-xs font-semibold uppercase tracking-wider text-base-content/70'>
						Column Policies
					</h4>
					<div class='grid grid-cols-1 sm:grid-cols-2 gap-2.5'>
						{/* Creation Policy */}
						<div class='p-3 rounded-xl bg-base-200/50 border border-base-300 flex flex-col gap-1'>
							<span class='text-[11px] font-semibold text-base-content/60 uppercase'>
								Task Creation
							</span>
							<div class='flex items-center gap-1.5 flex-wrap'>
								<span
									class={`badge badge-xs font-medium ${
										isCreatable ? 'badge-success' : 'badge-ghost'
									}`}
								>
									{isCreatable ? 'Allowed' : 'Restricted'}
								</span>
								{isDefaultCreate && (
									<span class='badge badge-xs badge-outline text-[10px]'>
										Default
									</span>
								)}
							</div>
							<p class='text-[11px] text-base-content/70 mt-0.5'>
								{isCreatable
									? isDefaultCreate
										? 'Initial column for newly created tasks.'
										: 'New tasks can be created directly in this column.'
									: 'Tasks cannot be created directly here.'}
							</p>
						</div>

						{/* Editable Policy */}
						<div class='p-3 rounded-xl bg-base-200/50 border border-base-300 flex flex-col gap-1'>
							<span class='text-[11px] font-semibold text-base-content/60 uppercase'>
								Specification Editing
							</span>
							<div class='flex items-center gap-1.5'>
								<span
									class={`badge badge-xs font-medium ${
										isEditable ? 'badge-info' : 'badge-warning'
									}`}
								>
									{isEditable ? 'Editable' : 'Locked'}
								</span>
							</div>
							<p class='text-[11px] text-base-content/70 mt-0.5'>
								{isEditable
									? 'Full task specification body is editable.'
									: 'Body is locked; only append notes are allowed.'}
							</p>
						</div>
					</div>
				</div>

				{/* Global AI System Prompt (MCP) if available */}
				{mcpInstructions && (
					<details class='collapse collapse-arrow bg-base-200/40 border border-base-300 rounded-xl text-left'>
						<summary class='collapse-title text-xs font-semibold uppercase tracking-wider text-base-content/70 py-2.5 px-3.5 min-h-0'>
							Global MCP System Prompt
						</summary>
						<div class='collapse-content border-t border-base-200 px-3.5 pt-2.5 pb-3 font-sans text-xs/relaxed whitespace-pre-wrap text-base-content/80'>
							{mcpInstructions}
						</div>
					</details>
				)}

				{/* Modal Footer / Actions */}
				<div class='modal-action pt-1 mt-0'>
					<button
						type='button'
						class='btn btn-sm btn-ghost'
						onClick={closeModal}
						data-testid='column-modal-close'
					>
						Close
					</button>
				</div>
			</div>
		</div>
	)
}
