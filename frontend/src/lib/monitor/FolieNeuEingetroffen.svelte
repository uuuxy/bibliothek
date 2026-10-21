<script>
	import { Sparkles } from '@lucide/svelte';
	import BuchCover from '../components/ui/BuchCover.svelte';

	/** @type {{ titel: import('./monitorTakt.svelte.js').MonitorTitel[], coverIndex: number }} */
	let { titel, coverIndex } = $props();
</script>

<!-- max-w-5xl (1024 px): zehn Cover in einer Reihe — 128 + 9 × 80 px plus neun 16-px-Lücken
     sind 992 px. In max-w-4xl (896 px) rutscht das zehnte Cover allein in eine zweite Zeile. -->
<div class="flex flex-col items-center gap-8 w-full max-w-5xl">
	<span class="text-sm font-medium text-primary"
		><Sparkles class="h-4 w-4" aria-hidden="true" /> Neu eingetroffen</span
	>
	{#if titel.length > 0}
		<div class="flex gap-4 items-end justify-center flex-wrap">
			{#each titel as book, i (i)}
				<!-- 60 % Deckkraft: Ohne Cover steht der Titel auf dem Platzhalter, und abgeblendet
				     braucht er noch 4,5:1 (WCAG 1.4.3, Gate barrierefreiheit-axe). -->
				<div
					class="flex flex-col items-center gap-2 transition-all duration-500"
					class:scale-110={i === coverIndex}
					class:opacity-60={i !== coverIndex}
				>
					<div class="transition-all duration-500 {i === coverIndex ? 'w-32' : 'w-20'}">
						<BuchCover
							coverUrl={book.cover_url}
							isbn={book.isbn}
							titel={book.titel}
							groesse="kachel"
							nurGespeichert
							klasse="shadow-lg"
						/>
					</div>
					{#if i === coverIndex}
						<div class="text-center max-w-32">
							<p class="text-sm font-bold leading-tight text-on-surface truncate">{book.titel}</p>
							{#if book.autor}
								<p class="text-xs text-on-surface-variant truncate">{book.autor}</p>
							{/if}
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{:else}
		<p class="text-on-surface-variant">Keine neuen Medien</p>
	{/if}
</div>
