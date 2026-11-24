<script>
	import { onMount } from 'svelte';
	import Ladekreis from '../components/ui/Ladekreis.svelte';
	import { apiFetch } from '../apiFetch.js';
	import { FOLIEN, FOLIE_MS, HEUTE, MonitorTakt, beschriftung } from './monitorTakt.svelte.js';
	import FolieLmfHeute from './FolieLmfHeute.svelte';
	import FolieBuchDesMonats from './FolieBuchDesMonats.svelte';
	import FolieNeuEingetroffen from './FolieNeuEingetroffen.svelte';
	import FolieBeliebt from './FolieBeliebt.svelte';

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

	// Die Folien sind im Raster des Fernsehers gezeichnet (864 × 450 innerhalb des Rands) und
	// wachsen mit dem Bildschirm, Google TV: „This ensures all elements can be resized
	// proportionally for HD or 4K screens“.
	let breite = $state(0);
	let hoehe = $state(0);
	const faktor = $derived(breite > 0 && hoehe > 0 ? Math.min(breite / 864, hoehe / 450) : 1);

	onMount(() => {
		takt.start();
		return () => takt.stop();
	});
</script>

<!-- Der Monitor steht im dunklen Schema (styles/rollen.css): Die Rollen tragen hier ihre
     dunklen Töne. -->
<main
	class="schema-dunkel fixed inset-0 bg-surface text-on-surface flex flex-col overflow-hidden select-none"
>
	<!-- Folienpunkte -->
	<div class="absolute top-4 left-1/2 -translate-x-1/2 flex gap-2 z-10">
		{#each FOLIEN as name, i (name)}
			<!-- Die Folie „Heute" hat nur an einem Tag des Plans einen Punkt. -->
			{#if i !== HEUTE || takt.slides?.lmf_heute}
				<button
					onclick={() => takt.springeZu(i)}
					aria-label="{beschriftung(i, takt.slides)} anzeigen"
					class="rounded-full transition-all duration-300 cursor-pointer {takt.folie === i
						? 'bg-on-surface w-6 h-2'
						: 'bg-outline w-2 h-2'}"
				></button>
			{/if}
		{/each}
	</div>

	<!-- Folie -->
	<div class="flex-1 flex px-8 py-16">
		<div
			class="flex-1 min-w-0 flex items-center justify-center"
			bind:clientWidth={breite}
			bind:clientHeight={hoehe}
		>
			{#if !takt.slides}
				<Ladekreis size="lg" />
			{:else}
				<div style:zoom={faktor} class="flex h-112.5 w-216 shrink-0 items-center justify-center">
					{#if takt.folie === HEUTE}
						{#if takt.slides.lmf_heute}<FolieLmfHeute heute={takt.slides.lmf_heute} />{/if}
					{:else if takt.folie === 1}
						<FolieBuchDesMonats titel={takt.slides.buch_des_monats} />
					{:else if takt.folie === 2}
						<FolieNeuEingetroffen
							titel={takt.slides.neu_eingetroffen}
							coverIndex={takt.coverIndex}
						/>
					{:else}
						<FolieBeliebt titel={takt.slides.beliebt} />
					{/if}
				</div>
			{/if}
		</div>
	</div>

	<!-- Beschriftung -->
	<div
		class="bg-surface-container-high px-6 py-3 flex items-center justify-between text-body-small-emphasized text-on-surface-variant tracking-wide"
	>
		<span data-testid="monitor-folie">{beschriftung(takt.folie, takt.slides)}</span>
		<span>Schulbibliothek</span>
	</div>

	<!-- Fortschrittsbalken: läuft genau eine Folie lang, dieselbe Zahl wie der Takt -->
	{#key takt.lauf}
		<div class="h-1 bg-surface-container-highest">
			<div class="h-full bg-primary progress-bar" style:animation-duration="{FOLIE_MS}ms"></div>
		</div>
	{/key}
</main>

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
