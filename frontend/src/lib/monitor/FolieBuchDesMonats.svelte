<script>
	import Icon from '../components/ui/Icon.svelte';
	import BuchCover from '../components/ui/BuchCover.svelte';

	/** @type {{ titel: import('./monitorTakt.svelte.js').MonitorTitel | null }} */
	let { titel } = $props();
</script>

<div class="flex flex-col items-center text-center gap-6 max-w-sm">
	<span class="text-sm font-medium text-primary"><Icon name="star" imText /> Buch des Monats</span>
	{#if titel}
		<div class="w-48">
			<BuchCover
				coverUrl={titel.cover_url}
				isbn={titel.isbn}
				titel={titel.titel}
				groesse="kachel"
				nurGespeichert
				klasse="shadow-level3 ring-4 ring-primary/30"
				dekorativ
			/>
		</div>
		<div>
			<h2 class="text-3xl font-extrabold leading-tight">{titel.titel}</h2>
			{#if titel.autor}
				<p class="text-on-surface-variant mt-2 text-lg">{titel.autor}</p>
			{/if}
		</div>
	{:else}
		<p class="text-on-surface-variant">Kein Buch verfügbar</p>
	{/if}
</div>
