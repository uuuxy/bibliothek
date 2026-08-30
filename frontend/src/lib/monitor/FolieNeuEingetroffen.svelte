<script>
	import { Sparkles } from '@lucide/svelte';
	import { coverSrc } from '../utils/coverSrc.js';

	/** @type {{ titel: import('./monitorTakt.svelte.js').MonitorTitel[], coverIndex: number }} */
	let { titel, coverIndex } = $props();
</script>

<!-- max-w-5xl (1024 px): zehn Cover in einer Reihe — 128 + 9 × 80 px plus neun 16-px-Lücken
     sind 992 px. Mit max-w-4xl (896 px) rutschte das zehnte Cover allein in eine zweite Zeile. -->
<div class="flex flex-col items-center gap-8 w-full max-w-5xl">
	<span class="text-sm font-medium text-cyan-400"
		><Sparkles class="h-4 w-4" aria-hidden="true" /> Neu eingetroffen</span
	>
	{#if titel.length > 0}
		<div class="flex gap-4 items-end justify-center flex-wrap">
			{#each titel as book, i (i)}
				<div
					class="flex flex-col items-center gap-2 transition-all duration-500"
					class:scale-110={i === coverIndex}
					class:opacity-50={i !== coverIndex}
				>
					{#if coverSrc(book.cover_url, book.isbn)}
						<img
							src={coverSrc(book.cover_url, book.isbn)}
							alt="Cover"
							class="rounded-xl shadow-lg object-cover transition-all duration-500
                           {i === coverIndex ? 'w-32 h-44' : 'w-20 h-28'}"
						/>
					{:else}
						<div
							class="rounded-xl bg-slate-700 flex items-center justify-center transition-all duration-500
                              {i === coverIndex ? 'w-32 h-44' : 'w-20 h-28'}"
						>
							<span
								class="{i === coverIndex ? 'text-2xl' : 'text-base'} text-slate-500 font-extrabold"
							>
								{book.titel.charAt(0)}
							</span>
						</div>
					{/if}
					{#if i === coverIndex}
						<div class="text-center max-w-32">
							<p class="text-sm font-bold leading-tight text-white truncate">{book.titel}</p>
							{#if book.autor}
								<p class="text-xs text-slate-400 truncate">{book.autor}</p>
							{/if}
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{:else}
		<p class="text-slate-500">Keine neuen Medien</p>
	{/if}
</div>
