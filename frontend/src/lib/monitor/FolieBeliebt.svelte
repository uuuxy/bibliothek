<script>
	import { Flame } from '@lucide/svelte';
	import BuchCover from '../components/ui/BuchCover.svelte';

	/** @type {{ titel: import('./monitorTakt.svelte.js').MonitorTitel[] }} */
	let { titel } = $props();
</script>

<div class="flex flex-col items-center gap-6 w-full max-w-lg">
	<span class="text-sm font-medium text-primary"
		><Flame class="h-4 w-4" aria-hidden="true" /> Beliebt diese Woche</span
	>
	{#if titel.length > 0}
		<ol class="w-full flex flex-col gap-3">
			{#each titel as book, i (i)}
				<li class="flex items-center gap-4 bg-surface-container rounded-2xl p-3 shadow-md">
					<span class="text-2xl font-black w-8 text-center text-on-surface-variant">#{i + 1}</span>
					<BuchCover
						coverUrl={book.cover_url}
						isbn={book.isbn}
						titel={book.titel}
						groesse="liste"
						nurGespeichert
					/>
					<div class="flex-1 min-w-0">
						<p class="font-bold truncate">{book.titel}</p>
						{#if book.autor}
							<p class="text-xs text-on-surface-variant truncate">{book.autor}</p>
						{/if}
					</div>
				</li>
			{/each}
		</ol>
	{:else}
		<p class="text-on-surface-variant">Keine Daten verfügbar</p>
	{/if}
</div>
