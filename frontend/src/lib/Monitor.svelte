<script>
	import { onMount } from 'svelte';
	import { apiFetch } from './apiFetch.js';
	import { FOLIEN, FOLIE_MS, MonitorTakt } from './monitor/monitorTakt.svelte.js';
	import FolieBuchDesMonats from './monitor/FolieBuchDesMonats.svelte';
	import FolieNeuEingetroffen from './monitor/FolieNeuEingetroffen.svelte';
	import FolieBeliebt from './monitor/FolieBeliebt.svelte';

	// Anzeige des Flur-Monitors — ohne Anmeldung, ohne Bedienung. Der Takt (Folienwechsel,
	// Nachladen, Neuversuch) lebt in monitor/monitorTakt.svelte.js und ist dort mit
	// gestellter Uhr geprüft; hier steht nur, was zu sehen ist.
	const takt = new MonitorTakt(async () => {
		try {
			const res = await apiFetch('/api/monitor/slides');
			return res.ok ? await res.json() : null;
		} catch {
			return null; // Netz weg: Der alte Stand bleibt stehen, der Takt versucht es wieder.
		}
	});

	onMount(() => {
		takt.start();
		return () => takt.stop();
	});
</script>

<div class="fixed inset-0 bg-slate-900 text-white flex flex-col overflow-hidden select-none">
	<!-- Folienpunkte -->
	<div class="absolute top-4 left-1/2 -translate-x-1/2 flex gap-2 z-10">
		{#each FOLIEN as name, i (name)}
			<button
				onclick={() => takt.springeZu(i)}
				aria-label="{name} anzeigen"
				class="rounded-full transition-all duration-300 cursor-pointer {takt.folie === i
					? 'bg-white w-6 h-2'
					: 'bg-slate-600 w-2 h-2'}"
			></button>
		{/each}
	</div>

	<!-- Folie -->
	<div class="flex-1 flex items-center justify-center px-8 py-16">
		{#if !takt.slides}
			<div class="text-slate-500 text-xl animate-pulse">Lade Daten …</div>
		{:else if takt.folie === 0}
			<FolieBuchDesMonats titel={takt.slides.buch_des_monats} />
		{:else if takt.folie === 1}
			<FolieNeuEingetroffen titel={takt.slides.neu_eingetroffen} coverIndex={takt.coverIndex} />
		{:else}
			<FolieBeliebt titel={takt.slides.beliebt} />
		{/if}
	</div>

	<!-- Beschriftung -->
	<div
		class="bg-slate-800 px-6 py-3 flex items-center justify-between text-xs text-slate-300 font-semibold tracking-wide"
	>
		<span data-testid="monitor-folie">{FOLIEN[takt.folie]}</span>
		<span class="text-slate-600">Schulbibliothek</span>
	</div>

	<!-- Fortschrittsbalken: läuft genau eine Folie lang, dieselbe Zahl wie der Takt -->
	{#key takt.lauf}
		<div class="h-1 bg-slate-800">
			<div class="h-full bg-slate-400 progress-bar" style:animation-duration="{FOLIE_MS}ms"></div>
		</div>
	{/key}
</div>

<style>
	@keyframes progress {
		from {
			width: 0%;
		}
		to {
			width: 100%;
		}
	}
	.progress-bar {
		animation: progress linear infinite;
	}
</style>
