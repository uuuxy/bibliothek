<script>
	import { Flame } from '@lucide/svelte';
	import { coverSrc } from '../utils/coverSrc.js';

	/** @type {{ titel: import('./monitorTakt.svelte.js').MonitorTitel[] }} */
	let { titel } = $props();
</script>

<div class="flex flex-col items-center gap-6 w-full max-w-lg">
	<span class="text-sm font-medium text-rose-400"
		><Flame class="h-4 w-4" aria-hidden="true" /> Beliebt diese Woche</span
	>
	{#if titel.length > 0}
		<ol class="w-full flex flex-col gap-3">
			{#each titel as book, i (i)}
				<li class="flex items-center gap-4 bg-slate-800/60 rounded-2xl p-3 shadow-md">
					<span class="text-2xl font-black w-8 text-center text-slate-500">#{i + 1}</span>
					{#if coverSrc(book.cover_url, book.isbn)}
						<img
							src={coverSrc(book.cover_url, book.isbn)}
							alt="Cover"
							class="w-12 h-16 object-cover rounded-xl shadow"
						/>
					{:else}
						<div class="w-12 h-16 rounded-xl bg-slate-700 flex items-center justify-center">
							<span class="text-lg font-extrabold text-slate-500">{book.titel.charAt(0)}</span>
						</div>
					{/if}
					<div class="flex-1 min-w-0">
						<p class="font-bold truncate">{book.titel}</p>
						{#if book.autor}
							<p class="text-xs text-slate-400 truncate">{book.autor}</p>
						{/if}
					</div>
				</li>
			{/each}
		</ol>
	{:else}
		<p class="text-slate-500">Keine Daten verfügbar</p>
	{/if}
</div>
