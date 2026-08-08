<script>
	import { onMount } from 'svelte';
	import { apiGet } from '../../apiFetch.js';
	import BestellDetail from './BestellDetail.svelte';
	import BestellHistorieTabelle from './BestellHistorieTabelle.svelte';
	import { orderStore } from '../../stores/orderStore.svelte.js';
	import { Clock } from '@lucide/svelte';

	/** @type {any[]} */
	let bestellungen = $state([]);
	let loading = $state(true);
	/** @type {string|null} Bestellung, deren Detailansicht offen ist (null = Liste). */
	let geoeffneteId = $state(null);

	/** @type {{gesamt: number, gesamtbetrag: number, gesamt_exemplare: number, offene_bestaetigungen: number}} */
	let uebersicht = $state({
		gesamt: 0,
		gesamtbetrag: 0,
		gesamt_exemplare: 0,
		offene_bestaetigungen: 0
	});

	async function ladeBestellungen() {
		// Zwei Anfragen mit Absicht: Die Liste ist auf die neuesten Bestellungen gedeckelt
		// (sonst 2,45 MB und rund vier Sekunden auf einer gewachsenen Datenbank), die
		// Kennzahlen im Kopf zählen aber weiterhin ALLE. Würden sie aus den geladenen Zeilen
		// gerechnet, stünde dort nach dem Deckeln eine zu kleine Zahl — die aussieht wie eine
		// Gesamtsumme.
		const [liste, summen] = await Promise.all([
			apiGet('/api/bestellhistorie'),
			apiGet('/api/bestellhistorie/uebersicht')
		]);
		bestellungen = liste || [];
		if (summen) uebersicht = summen;
		loading = false;
	}

	onMount(ladeBestellungen);

	let gesamtsumme = $derived(uebersicht.gesamtbetrag);
	let gesamtExemplare = $derived(uebersicht.gesamt_exemplare);
	// Serverseitig gezählt: Eine wartende Bestellung darf nicht deshalb unsichtbar bleiben,
	// weil sie hinter dem Listen-Limit liegt.
	let offeneBestaetigungen = $derived(uebersicht.offene_bestaetigungen);
	let gekappt = $derived(uebersicht.gesamt > bestellungen.length);

	/** @param {number} n */
	function euro(n) {
		return n.toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) + ' €';
	}

	/** @param {string} iso */
	function datum(iso) {
		return new Date(iso).toLocaleDateString('de-DE', {
			day: '2-digit',
			month: '2-digit',
			year: 'numeric'
		});
	}

	// Im Chip zählt Kürze: „05.08." genügt neben dem Wort „Bestätigt", das Jahr steht
	// bereits in der Datumsspalte derselben Zeile.
	/** @param {string} iso */
	function kurzdatum(iso) {
		return new Date(iso).toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit' });
	}

	/** @param {string} id */
	function oeffne(id) {
		geoeffneteId = id;
	}

	/**
	 * Zurück aus dem Detail — und die Liste neu laden.
	 *
	 * Das Nachladen ist kein Beiwerk: Im Detail lässt sich die Bestellung bestätigen und
	 * ein neuer Bestätigungs-Link erzeugen (BestellStatusBlock). Ohne das Neuladen stünde
	 * in der Liste danach der alte Stand, und der Statuszähler im Kopf ebenso.
	 */
	function zurueck() {
		geoeffneteId = null;
		ladeBestellungen();
	}
</script>

<!-- Liste ODER Detail, nicht beides: dieselbe Bauart wie der Wareneingang im Reiter
     nebenan (BestellWorkspace). Kein eigener Router-Eintrag — die Bestellung ist ein
     Zustand DIESES Reiters, und der Zurück-Knopf führt an die Stelle zurück, an der
     man war. -->
{#if geoeffneteId}
	<BestellDetail bestellungId={geoeffneteId} onBack={zurueck} />
{:else}
	<div class="space-y-6">
		<div class="flex items-center justify-between border-b border-slate-200 pb-4">
			<div>
				<h2 class="text-base font-bold text-slate-800">Bestellhistorie</h2>
				<p class="text-sm text-slate-500 mt-0.5">
					Alle aufgegebenen Bestellungen — automatisch erfasst beim Bestellen
				</p>
				<!-- Nur wenn wirklich etwas aussteht. „Alles bestätigt" jeden Tag zu lesen, wäre
			     dieselbe Zeile ohne Nachricht — auffallen soll die Abweichung. Wer den Satz
			     sieht, weiß ohne Scrollen, dass in der Statusspalte etwas auf ihn wartet. -->
				{#if offeneBestaetigungen > 0}
					<p class="mt-2 flex items-center gap-1.5 text-sm font-medium text-amber-700">
						<Clock size={15} aria-hidden="true" />
						{offeneBestaetigungen === 1
							? '1 Bestellung wartet noch auf die Bestätigung des Händlers'
							: `${offeneBestaetigungen} Bestellungen warten noch auf die Bestätigung des Händlers`}
					</p>
				{/if}
			</div>
			<!-- Ohne Preiserfassung ist "Gesamtausgaben 0,00 €" keine Auskunft, sondern eine
		     falsche: Die Schule hat ausgegeben, nur steht es nirgends. Dann lieber die Zahl
		     nennen, die stimmt. -->
			{#if bestellungen.length > 0}
				<div class="text-right">
					{#if orderStore.preiseErfassen}
						<div class="text-xs text-slate-400 font-semibold">Gesamtausgaben</div>
						<div class="text-2xl font-black text-slate-800">{euro(gesamtsumme)}</div>
					{:else}
						<div class="text-xs text-slate-400 font-semibold">Bestellte Exemplare</div>
						<div class="text-2xl font-black text-slate-800">{gesamtExemplare}</div>
					{/if}
				</div>
			{/if}
		</div>

		{#if loading}
			<div class="py-16 text-center text-slate-400 text-base animate-pulse">
				Lade Bestellhistorie…
			</div>
		{:else if bestellungen.length === 0}
			<div class="py-16 text-center text-slate-400 text-base">
				Noch keine Bestellungen aufgegeben.<br />
				<span class="text-sm">Bestellungen werden hier automatisch gespeichert.</span>
			</div>
		{:else}
			<BestellHistorieTabelle {bestellungen} {euro} {datum} {kurzdatum} onOeffnen={oeffne} />

			<!-- Ehrlich sagen, dass die Liste nicht alles zeigt. Ohne den Satz sucht jemand eine
		     ältere Bestellung, findet sie nicht und hält sie für gelöscht. -->
			{#if gekappt}
				<p class="text-center text-xs text-slate-400">
					Neueste {bestellungen.length} von {uebersicht.gesamt} Bestellungen — ältere stehen im Bericht.
				</p>
			{/if}
		{/if}
	</div>
{/if}
