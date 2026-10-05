import { User, X } from 'lucide-preact'
import { useLayoutEffect } from 'preact/hooks'
import type { Column } from '../../schemas/models'
import { activeColumnDetailId, config } from '../../state/store'
import { t, tf, tParts } from '../../utils/i18n'

export function ColumnDetailModal() {
	const columnId = activeColumnDetailId.value
	const column = config.value.board.columns.find((c) => c.id === columnId)
	if (!columnId || !column) {
		return null
	}
	return <ColumnDetailDialog key={column.id} column={column} />
}

function ColumnDetailDialog({ column }: { column: Column }) {
	const boardCfg = config.value.board
	const mcpInstructions = config.value.mcp?.instructions

	function closeModal() {
		activeColumnDetailId.value = null
	}

	useLayoutEffect(() => {
		function handleKeyDown(e: KeyboardEvent) {
			if (e.key === 'Escape') {
				closeModal()
			}
		}
		window.addEventListener('keydown', handleKeyDown)
		return () => window.removeEventListener('keydown', handleKeyDown)
	}, [])

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
			aria-label={tf('arial_column_details', { name: column.name })}
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
							{tf('column_status_id', { id: column.id })}
						</span>
					</div>
					<button
						type='button'
						class='btn btn-sm btn-circle btn-ghost text-base-content/70 hover:text-base-content'
						onClick={closeModal}
						aria-label={t('arial_close_modal')}
						data-testid='column-modal-close-btn'
					>
						<X class='size-4' />
					</button>
				</div>

				{/* Role & Handled By Section */}
				<div class='flex flex-col gap-1.5'>
					<h4 class='text-xs font-semibold uppercase tracking-wider text-base-content/70'>
						{t('column_handled_by')}
					</h4>
					{column.handled_by ? (
						<div
							class='flex items-start gap-3 p-3 rounded-xl bg-base-200/60 border border-base-300'
							data-testid='column-modal-handled-by'
						>
							<div class='size-8 rounded-lg bg-primary/10 text-primary flex items-center justify-center shrink-0 mt-0.5'>
								<User class='size-4' />
							</div>
							<div class='flex-1 min-w-0'>
								<div class='flex items-center gap-2'>
									<span class='font-bold text-sm text-base-content'>
										{column.handled_by}
									</span>
									<span class='badge badge-xs badge-primary font-medium'>
										{t('column_assigned')}
									</span>
								</div>
								<p class='text-xs text-base-content/70 mt-0.5'>
									{tParts('column_assigned_text', {
										handler: (
											<span class='font-semibold text-base-content'>
												{column.handled_by}
											</span>
										),
									})}
								</p>
							</div>
						</div>
					) : (
						<div
							class='p-3 rounded-xl bg-base-200/40 border border-base-200 text-xs text-base-content/70 flex items-center gap-2'
							data-testid='column-modal-handled-by'
						>
							<span class='badge badge-xs badge-ghost'>
								{t('column_unassigned')}
							</span>
							<span>{t('column_unassigned_text')}</span>
						</div>
					)}
				</div>

				{/* Workflow Instructions & Prompt Section */}
				<div class='flex flex-col gap-1.5'>
					<div class='flex items-center justify-between'>
						<h4 class='text-xs font-semibold uppercase tracking-wider text-base-content/70'>
							{t('column_instructions')}
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
							{tParts('column_no_instructions', {
								instructions: (
									<code class='font-mono text-xs bg-base-300 px-1 py-0.5 rounded'>
										instructions = "..."
									</code>
								),
								section: (
									<code class='font-mono text-xs bg-base-300 px-1 py-0.5 rounded'>
										[[board.columns]]
									</code>
								),
							})}
						</div>
					)}
				</div>

				{/* Board Column Policies Grid */}
				<div class='flex flex-col gap-1.5'>
					<h4 class='text-xs font-semibold uppercase tracking-wider text-base-content/70'>
						{t('column_policies')}
					</h4>
					<div class='grid grid-cols-1 sm:grid-cols-2 gap-2.5'>
						{/* Creation Policy */}
						<div class='p-3 rounded-xl bg-base-200/50 border border-base-300 flex flex-col gap-1'>
							<span class='text-[11px] font-semibold text-base-content/60 uppercase'>
								{t('column_task_creation')}
							</span>
							<div class='flex items-center gap-1.5 flex-wrap'>
								<span
									class={`badge badge-xs font-medium ${
										isCreatable ? 'badge-success' : 'badge-ghost'
									}`}
								>
									{isCreatable
										? t('column_creation_allowed')
										: t('column_creation_restricted')}
								</span>
								{isDefaultCreate && (
									<span class='badge badge-xs badge-outline text-[10px]'>
										{t('column_creation_default')}
									</span>
								)}
							</div>
							<p class='text-[11px] text-base-content/70 mt-0.5'>
								{isCreatable
									? isDefaultCreate
										? t('column_creation_default_text')
										: t('column_creation_allowed_text')
									: t('column_creation_restricted_text')}
							</p>
						</div>

						{/* Editable Policy */}
						<div class='p-3 rounded-xl bg-base-200/50 border border-base-300 flex flex-col gap-1'>
							<span class='text-[11px] font-semibold text-base-content/60 uppercase'>
								{t('column_spec_editing')}
							</span>
							<div class='flex items-center gap-1.5'>
								<span
									class={`badge badge-xs font-medium ${
										isEditable ? 'badge-info' : 'badge-warning'
									}`}
								>
									{isEditable ? t('column_editable') : t('column_locked')}
								</span>
							</div>
							<p class='text-[11px] text-base-content/70 mt-0.5'>
								{isEditable
									? t('column_editable_text')
									: t('column_locked_text')}
							</p>
						</div>
					</div>
				</div>

				{/* Global AI System Prompt (MCP) if available */}
				{mcpInstructions && (
					<details class='collapse collapse-arrow bg-base-200/40 border border-base-300 rounded-xl text-left'>
						<summary class='collapse-title text-xs font-semibold uppercase tracking-wider text-base-content/70 py-2.5 px-3.5 min-h-0'>
							{t('column_mcp_prompt')}
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
						{t('close')}
					</button>
				</div>
			</div>
		</div>
	)
}
