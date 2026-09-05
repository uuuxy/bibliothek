<!-- @component PortalLmfPlan — der LMF-Plan im Kollegiums-Portal: dieselbe Tabelle wie
     in der Bibliothek, für alle gleich, immer der aktuelle Stand (kein abgelegtes PDF).
     Kein Bezug auf Klassenleitungen: Auch Fachlehrer gehen mit ihren Klassen zum
     Büchertausch, und nicht jede Klasse hat eine hinterlegte Adresse (Peter, 05.09.2026). -->
<script>
	import { onMount } from 'svelte';
	import { CalendarDays, Printer } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import LmfPlanTabelle from '../lmfplan/LmfPlanTabelle.svelte';
	import * as dienst from '../../lmfplanDienst.js';

	/** @type {any[]} */
	let termine = $state([]);
	let laedt = $state(true);
	let fehler = $state('');

	onMount(async () => {
		try {
			termine = (await dienst.ladePlan()).termine;
		} catch (e) {
			fehler = `${e}`;
		} finally {
			laedt = false;
		}
	});
</script>

<div class="flex flex-wrap items-center justify-between gap-3 pb-2">
	<p class="text-sm text-on-surface-variant max-w-2xl">
		Wann welche Klasse ihre Schulbücher zurückgibt oder neue bekommt — der Plan der Bibliothek,
		immer auf dem aktuellen Stand.
	</p>
	<Button variant="secondary" onclick={() => dienst.ladePdf()} disabled={termine.length === 0}>
		<Printer class="h-4 w-4" aria-hidden="true" />
		Als PDF
	</Button>
</div>

{#if laedt}
	<div class="py-12 flex justify-center items-center">
		<div
			class="w-8 h-8 border-2 border-t-primary border-surface-container-high rounded-full animate-spin"
		></div>
	</div>
{:else if fehler}
	<p class="py-8 text-center text-sm text-error">{fehler}</p>
{:else if termine.length === 0}
	<div class="py-12 text-center space-y-3 animate-fade-in">
		<div
			class="w-16 h-16 rounded-full bg-surface-container-low border border-outline-variant flex items-center justify-center text-on-surface-variant mx-auto"
		>
			<CalendarDays class="h-8 w-8" aria-hidden="true" />
		</div>
		<h3 class="font-bold text-on-surface">Noch kein Plan für dieses Schuljahr</h3>
		<p class="text-xs text-on-surface-variant max-w-sm mx-auto">
			Sobald die Bibliothek die Termine einträgt, stehen sie hier.
		</p>
	</div>
{:else}
	<LmfPlanTabelle {termine} />
{/if}
