<script>
	import Icon from '../components/ui/Icon.svelte';
	import BuchCover from '../components/ui/BuchCover.svelte';

	/** @type {{ titel: import('./monitorTakt.svelte.js').MonitorTitel[], coverIndex: number }} */
	let { titel, coverIndex } = $props();
</script>

<!-- Zehn Cover in einer Reihe auf der Folie (864 breit, Monitor.svelte): das gezeigte 112, neun
     weitere je 64 und neun Lücken je 12 sind 796; die Zeilen unter dem gezeigten dürfen bis 144
     breit werden. Mit den früheren 128, 80 und 16 wären es 992, das zehnte Cover rutschte in eine
     zweite Zeile. -->
<div class="flex flex-col items-center gap-8 w-full">
	<span class="text-sm font-medium text-primary"
		><Icon name="auto_awesome" imText /> Neu eingetroffen</span
	>
	{#if titel.length > 0}
		<div class="flex gap-3 items-end justify-center flex-wrap">
			{#each titel as book, i (i)}
				<!-- 60 % Deckkraft: Ohne Cover steht der Titel auf dem Platzhalter, und abgeblendet
				     braucht er noch 4,5:1 (WCAG 1.4.3, Gate barrierefreiheit-axe). -->
				<div
					class="flex flex-col items-center gap-2 transition-all duration-500"
					class:scale-110={i === coverIndex}
					class:opacity-60={i !== coverIndex}
				>
					<div class="transition-all duration-500 {i === coverIndex ? 'w-28' : 'w-16'}">
						<BuchCover
							coverUrl={book.cover_url}
							isbn={book.isbn}
							titel={book.titel}
							groesse="kachel"
							nurGespeichert
							klasse="shadow-level2"
							dekorativ
						/>
					</div>
					{#if i === coverIndex}
						<div class="text-center max-w-36">
							<p class="text-body-medium-emphasized leading-tight text-on-surface truncate">
								{book.titel}
							</p>
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
