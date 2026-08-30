<script>
	import { Star } from '@lucide/svelte';
	import { coverSrc } from '../utils/coverSrc.js';

	/** @type {{ titel: import('./monitorTakt.svelte.js').MonitorTitel | null }} */
	let { titel } = $props();
</script>

<div class="flex flex-col items-center text-center gap-6 max-w-sm">
	<span class="text-sm font-medium text-amber-400"
		><Star class="h-4 w-4" aria-hidden="true" /> Buch des Monats</span
	>
	{#if titel}
		{#if coverSrc(titel.cover_url, titel.isbn)}
			<img
				src={coverSrc(titel.cover_url, titel.isbn)}
				alt="Cover"
				class="w-48 h-64 object-cover rounded-2xl shadow-2xl ring-4 ring-amber-400/30"
			/>
		{:else}
			<div class="w-48 h-64 rounded-2xl bg-slate-700 flex items-center justify-center shadow-2xl">
				<span class="text-6xl font-extrabold text-slate-500">{titel.titel.charAt(0)}</span>
			</div>
		{/if}
		<div>
			<h2 class="text-3xl font-extrabold leading-tight">{titel.titel}</h2>
			{#if titel.autor}
				<p class="text-slate-400 mt-2 text-lg">{titel.autor}</p>
			{/if}
		</div>
	{:else}
		<p class="text-slate-500">Kein Buch verfügbar</p>
	{/if}
</div>
