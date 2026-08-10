<script>
	import { appState } from '$lib/store.svelte.js';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import Suchfeld from '../../../../lib/components/ui/Suchfeld.svelte';
	import { BookOpen, Plus, RefreshCw, Settings, Trash2 } from '@lucide/svelte';

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
		<Suchfeld
			bind:wert={appState.searchQuery}
			platzhalter="Titel, Autor oder ISBN eingeben …"
			etikett="Bücher durchsuchen"
			klasse="w-full sm:max-w-md"
		/>
	</div>

	<div class="flex flex-wrap items-center gap-2 sm:gap-3">
		{#if selectedCount > 0}
			<Button
				variant="secondary"
				onclick={onAssignClass}
				class="border-blue-100 bg-blue-50 text-blue-600 hover:bg-blue-100/60"
			>
				<BookOpen class="w-4 h-4" aria-hidden="true" />
				Klasse zuweisen ({selectedCount})
			</Button>
			<Button variant="danger" onclick={onDelete}>
				<Trash2 class="w-4 h-4" aria-hidden="true" />
				Löschen ({selectedCount})
			</Button>
		{/if}

		<Button variant="secondary" onclick={onRetryCovers} class="flex-1 sm:flex-none">
			<RefreshCw class="w-4 h-4 text-slate-500" aria-hidden="true" />
			Retry Cover
		</Button>

		<Button variant="secondary" onclick={onScan} class="flex-1 sm:flex-none">
			<Settings class="w-4 h-4 text-slate-500" aria-hidden="true" />
			Scanner
		</Button>

		<Button onclick={onCreateNew} class="mt-2 w-full sm:mt-0 sm:w-auto">
			<Plus class="w-4 h-4" aria-hidden="true" />
			Neues Buch
		</Button>
	</div>
</div>
