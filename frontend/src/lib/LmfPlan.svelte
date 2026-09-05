<script>
	import { onMount } from 'svelte';
	import { CalendarDays, Plus, Printer } from '@lucide/svelte';
	import PageShell from './components/layout/PageShell.svelte';
	import Button from './components/ui/Button.svelte';
	import LmfPlanTabelle from './components/lmfplan/LmfPlanTabelle.svelte';
	import LmfTerminDialog from './components/lmfplan/LmfTerminDialog.svelte';
	import { showToast } from '../inventur/lib/store.svelte.js';
	import { apiFetch } from './apiFetch.js';
	import * as dienst from './lmfplanDienst.js';

	// Der LMF-Plan der Bibliothek (Register, Entscheidung 3, 05.09.2026): Rückgabe- und
	// Ausgabetermine je Klasse, gepflegt statt in Excel. Neuer Plan startet leer; die
	// Seite nennt, welche Klassen noch keinen Rückgabe-Termin haben.
	/** @type {any[]} */
	let termine = $state([]);
	/** @type {string[]} */
	let ohneTermin = $state([]);
	/** @type {string[]} */
	let klassen = $state([]);
	let alle = $state(false);
	let laedt = $state(true);
	let dialogOffen = $state(false);
	/** @type {any} */
	let bearbeitet = $state(null);

	async function lade() {
		laedt = true;
		try {
			const plan = await dienst.ladePlan(alle);
			termine = plan.termine;
			ohneTermin = plan.ohne_rueckgabe_termin;
		} catch (e) {
			showToast(`${e}`, 'error');
		} finally {
			laedt = false;
		}
	}

	async function ladeKlassen() {
		try {
			const res = await apiFetch('/api/klassen');
			if (res.ok) klassen = (await res.json()) || [];
		} catch {
			klassen = [];
		}
	}

	/** @param {any} t */
	function oeffne(t) {
		bearbeitet = t;
		dialogOffen = true;
	}

	/** @param {any} t */
	async function speichere(t) {
		const erg = await dienst.speichereTermin(t);
		showToast(erg.meldung, erg.ok ? 'success' : 'error');
		if (erg.ok) {
			dialogOffen = false;
			await lade();
		}
	}

	/** @param {any} t */
	async function loesche(t) {
		if (!confirm(`Termin am ${dienst.datumKurz(t.datum)}, ${dienst.stundeText(t.stunde)} löschen?`))
			return;
		const erg = await dienst.loescheTermin(t.id);
		showToast(erg.meldung, erg.ok ? 'success' : 'error');
		if (erg.ok) await lade();
	}

	async function pdf() {
		try {
			await dienst.ladePdf(alle);
		} catch (e) {
			showToast(`${e}`, 'error');
		}
	}

	onMount(() => {
		lade();
		ladeKlassen();
	});
</script>

<PageShell>
	<div
		class="flex flex-wrap items-center justify-between gap-3 border-b border-outline-variant pb-4"
	>
		<p class="text-sm text-on-surface-variant max-w-2xl">
			Rückgabe- und Ausgabetermine je Klasse. Das Kollegium sieht denselben Plan in seinem Portal;
			der Rückgabe-Termin einer Klasse ist die Frist ihrer Schulbücher.
		</p>
		<div class="flex items-center gap-2 shrink-0">
			<Button variant="secondary" onclick={() => ((alle = !alle), lade())}>
				{alle ? 'Nur laufendes Schuljahr' : 'Ältere anzeigen'}
			</Button>
			<Button variant="secondary" onclick={pdf} disabled={termine.length === 0}>
				<Printer class="h-4 w-4" aria-hidden="true" />
				Als PDF
			</Button>
			<Button onclick={() => oeffne(null)}>
				<Plus class="h-4 w-4" aria-hidden="true" />
				Termin hinzufügen
			</Button>
		</div>
	</div>

	{#if !alle && ohneTermin.length > 0}
		<p class="mt-3 text-sm text-on-surface-variant" data-testid="lmf-ohne-termin">
			<span class="font-medium text-on-surface">Noch ohne Rückgabe-Termin:</span>
			{ohneTermin.join(', ')}
		</p>
	{/if}

	{#if laedt}
		<div class="py-12 flex justify-center items-center">
			<div
				class="w-8 h-8 border-2 border-t-primary border-surface-container-high rounded-full animate-spin"
			></div>
		</div>
	{:else if termine.length === 0}
		<div class="py-12 text-center space-y-3 animate-fade-in">
			<div
				class="w-16 h-16 rounded-full bg-surface-container-low border border-outline-variant flex items-center justify-center text-on-surface-variant mx-auto"
			>
				<CalendarDays class="h-8 w-8" aria-hidden="true" />
			</div>
			<h3 class="font-bold text-on-surface">Noch kein Termin eingetragen</h3>
			<p class="text-xs text-on-surface-variant max-w-sm mx-auto">
				Mit „Termin hinzufügen" beginnt der Plan — Klasse für Klasse, Abschlussklassen zuerst.
			</p>
		</div>
	{:else}
		<LmfPlanTabelle {termine} bearbeitbar onBearbeiten={oeffne} onLoeschen={loesche} />
	{/if}
</PageShell>

<LmfTerminDialog
	open={dialogOffen}
	termin={bearbeitet}
	{klassen}
	onclose={() => (dialogOffen = false)}
	onspeichern={speichere}
/>
