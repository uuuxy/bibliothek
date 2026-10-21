<script>
	import { Star } from '@lucide/svelte';
	import BuchCover from '../components/ui/BuchCover.svelte';

	/** @type {{ titel: import('./monitorTakt.svelte.js').MonitorTitel | null }} */
	let { titel } = $props();
</script>

<div class="flex flex-col items-center text-center gap-6 max-w-sm">
	<span class="text-sm font-medium text-primary"
		><Star class="h-4 w-4" aria-hidden="true" /> Buch des Monats</span
	>
	{#if titel}
		<div class="w-48">
			<BuchCover
				coverUrl={titel.cover_url}
				isbn={titel.isbn}
				titel={titel.titel}
				groesse="kachel"
				nurGespeichert
				klasse="shadow-2xl ring-4 ring-primary/30"
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
