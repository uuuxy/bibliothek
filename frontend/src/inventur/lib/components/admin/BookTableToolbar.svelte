<script>
	import { appState } from '$lib/store.svelte.js';

	/**
	 * @type {{
	 *   booksLength: number,
	 *   selectedCount: number,
	 *   onDelete: () => void,
	 *   onAssignClass: () => void,
	 *   onScan: () => void,
	 *   onCreateNew: () => void,
	 *   onRetryCovers: () => void
	 * }}
	 */
	let { booksLength, selectedCount, onDelete, onAssignClass, onScan, onCreateNew, onRetryCovers } =
		$props();
</script>

<div
	class="px-4 py-4 md:px-6 border-b border-slate-100 flex flex-col md:flex-row items-stretch md:items-center justify-between bg-white gap-4"
>
	<div class="flex flex-col sm:flex-row items-start sm:items-center gap-4 flex-1">
		<h2 class="text-lg font-bold text-slate-900 shrink-0">
			Bücher ({booksLength})
		</h2>
		<div class="relative w-full sm:max-w-md">
			<svg
				class="w-5 h-5 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
				/>
			</svg>
			<input
				type="text"
				placeholder="Suchen..."
				bind:value={appState.searchQuery}
				class="w-full pl-10 pr-4 py-2 bg-slate-50 border border-slate-200 rounded-xl text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500"
			/>
		</div>
	</div>

	<div class="flex flex-wrap items-center gap-2 sm:gap-3">
		{#if selectedCount > 0}
			<button
				onclick={onAssignClass}
				class="flex items-center gap-2 px-3 sm:px-4 py-2 rounded-xl text-sm font-semibold text-blue-600 bg-blue-50 border border-blue-100 hover:bg-blue-100/60 transition-colors cursor-pointer"
			>
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"
					/>
				</svg>
				Klasse zuweisen ({selectedCount})
			</button>
			<button
				onclick={onDelete}
				class="flex items-center gap-2 px-3 sm:px-4 py-2 rounded-xl text-sm font-semibold text-rose-600 bg-rose-50 border border-rose-100 hover:bg-rose-100/60 transition-colors cursor-pointer"
			>
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
					/>
				</svg>
				Löschen ({selectedCount})
			</button>
		{/if}

		<button
			onclick={onRetryCovers}
			class="flex-1 sm:flex-none justify-center flex items-center gap-2 px-3 sm:px-4 py-2 rounded-xl text-sm font-semibold text-slate-650 bg-white border border-slate-200 hover:bg-slate-50 transition-colors cursor-pointer"
		>
			<svg class="w-4 h-4 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0A8.003 8.003 0 015.59 15m13.828 0H15"
				/>
			</svg>
			Retry Cover
		</button>

		<button
			onclick={onScan}
			class="flex-1 sm:flex-none justify-center flex items-center gap-2 px-3 sm:px-4 py-2 rounded-xl text-sm font-semibold text-slate-650 bg-white border border-slate-200 hover:bg-slate-50 transition-colors cursor-pointer"
		>
			<svg class="w-4 h-4 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z"
				/>
			</svg>
			Scanner
		</button>

		<button
			onclick={onCreateNew}
			class="w-full sm:w-auto mt-2 sm:mt-0 justify-center flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-bold text-white bg-blue-600 hover:bg-blue-700 transition-all cursor-pointer shadow-xs"
		>
			<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
			</svg>
			Neues Buch
		</button>
	</div>
</div>
