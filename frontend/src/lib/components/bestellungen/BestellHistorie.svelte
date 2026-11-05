<script>
	import { onMount } from 'svelte';
	import { apiGet } from '../../apiFetch.js';
	import { abonniereNeuladen } from '../../liveEvents.js';
	import BestellDetail from './BestellDetail.svelte';
	import BestellHistorieTabelle from './BestellHistorieTabelle.svelte';
	import BestellHistorieKopf from './BestellHistorieKopf.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import SuchZustand from '../ui/SuchZustand.svelte';
	import { ShoppingBag } from '@lucide/svelte';
	import { MITTEL, MITTEL_REIHENFOLGE } from './mittel.js';
	import { formatEuro } from '../../utils/format.js';

	/** @type {any[]} */
	let bestellungen = $state([]);
	let laedt = $state(true);
	/** Leer heißt leer, ein Ladefehler heißt Ladefehler. */
	let ladefehler = $state('');
	// Nur die jüngste Anfrage schreibt die Liste: Nach zwei Filterwechseln entschiede sonst die
	// Reihenfolge der Antworten, welcher Topf unter dem Filter steht.
	let ladeNr = 0;
	/** @type {string|null} Der Topf der zuletzt geladenen Liste. */
	let geladenerTopf = null;
	/** @type {string|null} Bestellung, deren Detailansicht offen ist (null = Liste). */
	let geoeffneteId = $state(null);

	/** @type {{gesamt: number, gesamtbetrag: number, gesamt_exemplare: number, offene_bestaetigungen: number, nach_mittel: any[]}} */
	let uebersicht = $state({
		gesamt: 0,
		gesamtbetrag: 0,
		gesamt_exemplare: 0,
		offene_bestaetigungen: 0,
		nach_mittel: []
	});
	// Erst mit der ersten Antwort ist bekannt, ob etwas wartet.
	let standBekannt = $state(false);

	// Der Topf-Filter. Gefiltert wird SERVERSEITIG: Die Liste ist auf die neuesten 200
	// Bestellungen gedeckelt, ein Filter im Browser zeigte also nur, was davon übrig
	// bleibt. „ohne" sind die Alt-Bestellungen ohne eindeutige Zuordnung — genau sie
	// sucht, wer den Topf im Detail nachträgt.
	let mittel = $state('');
	const mittelFilter = [
		{ value: '', label: 'Alle Mittel' },
		...MITTEL_REIHENFOLGE.map((m) => ({
			value: m,
			label: `${MITTEL[m].label} (${MITTEL[m].traeger})`
		})),
		{ value: 'ohne', label: 'ohne Zuordnung' }
	];

	async function ladeBestellungen() {
		// Zwei Anfragen mit Absicht: Die Liste ist auf die neuesten Bestellungen gedeckelt
		// (sonst 2,45 MB und rund vier Sekunden auf einer gewachsenen Datenbank), die
		// Kennzahlen im Kopf zählen aber weiterhin ALLE. Würden sie aus den geladenen Zeilen
		// gerechnet, stünde dort nach dem Deckeln eine zu kleine Zahl — die aussieht wie eine
		// Gesamtsumme.
		const nr = ++ladeNr;
		const topf = mittel;
		try {
			const [liste, summen] = await Promise.all([
				apiGet('/api/bestellhistorie' + (topf ? `?mittel=${topf}` : '')),
				apiGet('/api/bestellhistorie/uebersicht')
			]);
			if (nr !== ladeNr) return true;
			bestellungen = liste || [];
			geladenerTopf = topf;
			if (summen) {
				uebersicht = summen;
				standBekannt = true;
			}
			ladefehler = '';
			return true;
		} catch (e) {
			// Scheitert nur das Auffrischen, bleibt die Liste stehen (die Meldung zeigt apiGet). Unter
			// einem anderen Filter stünde sonst die Liste des vorigen Topfes.
			if (nr === ladeNr && geladenerTopf !== topf) {
				bestellungen = [];
				ladefehler =
					/** @type {any} */ (e)?.message || 'Die Bestellhistorie konnte nicht geladen werden.';
			}
			return false;
		} finally {
			if (nr === ladeNr) laedt = false;
		}
	}

	// Hinter einer geöffneten Bestellung bleibt die Liste, wie sie ist: Der Rückweg lädt sie neu, und
	// die Bestellung holt sich ihren Stand selbst.
	onMount(() => {
		void ladeBestellungen();
		return abonniereNeuladen('bestellstand', () => (geoeffneteId ? true : ladeBestellungen()));
	});

	let gesamtsumme = $derived(uebersicht.gesamtbetrag);
	let gesamtExemplare = $derived(uebersicht.gesamt_exemplare);
	// Serverseitig gezählt: Eine wartende Bestellung darf nicht deshalb unsichtbar bleiben,
	// weil sie hinter dem Listen-Limit liegt.
	let offeneBestaetigungen = $derived(uebersicht.offene_bestaetigungen);
	// Gekappt heißt „die Liste zeigt nicht alles" — und das ist beim gefilterten Blick
	// die Zahl DIESES Topfes, nicht die aller Bestellungen. Sonst stünde unter einer
	// vollständigen Liste „Neueste 2 von 137".
	let gesamtImBlick = $derived(
		mittel === ''
			? uebersicht.gesamt
			: (uebersicht.nach_mittel?.find(
					(/** @type {any} */ t) => t.mittel === (mittel === 'ohne' ? '' : mittel)
				)?.gesamt ?? bestellungen.length)
	);
	let gekappt = $derived(gesamtImBlick > bestellungen.length);
	// Die Aufteilung im Kopf: Was steckt in den Gesamtzahlen je Topf? Nur Zeilen mit
	// Bestellungen — „Schülerbücherei: 0" sagt nichts, was die Zeile darüber nicht schon
	// sagt.
	let aufteilung = $derived(
		(uebersicht.nach_mittel ?? []).filter((/** @type {any} */ t) => t.gesamt > 0)
	);
	/** @param {string} wert */
	const topfName = (wert) =>
		wert && wert in MITTEL ? `${MITTEL[/** @type {any} */ (wert)].label}` : 'ohne Zuordnung';

	/** @param {string} iso */
	function datum(iso) {
		return new Date(iso).toLocaleDateString('de-DE', {
			day: '2-digit',
			month: '2-digit',
			year: 'numeric'
		});
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
     nebenan (Bestellwesen). Kein eigener Router-Eintrag — die Bestellung ist ein
     Zustand DIESES Reiters, und der Zurück-Knopf führt an die Stelle zurück, an der
     man war. -->
{#if geoeffneteId}
	<BestellDetail bestellungId={geoeffneteId} onBack={zurueck} />
{:else}
	<div class="space-y-6">
		<BestellHistorieKopf
			{offeneBestaetigungen}
			{standBekannt}
			{gesamtsumme}
			{gesamtExemplare}
			{aufteilung}
			euro={formatEuro}
			{topfName}
			{mittelFilter}
			zeigeKennzahlen={bestellungen.length > 0}
			bind:mittel
			onFilterWechsel={ladeBestellungen}
		/>

		{#if laedt}
			<div class="py-16 text-center text-on-surface-variant text-base animate-pulse">
				Lade Bestellhistorie…
			</div>
		{:else if ladefehler}
			<LadeFehler
				onerneut={ladeBestellungen}
				titel="Bestellhistorie nicht geladen"
				text={ladefehler}
			/>
		{:else if bestellungen.length === 0}
			<SuchZustand
				symbol={ShoppingBag}
				titel={mittel ? 'Keine Bestellungen in diesem Topf' : 'Noch keine Bestellungen aufgegeben'}
				hinweis={mittel
					? 'Den Filter aufheben, um alle zu sehen.'
					: 'Bestellungen werden hier automatisch gespeichert.'}
			/>
		{:else}
			<!-- Ehrlich sagen, dass die Liste nicht alles zeigt. Ohne den Satz sucht jemand eine
		     ältere Bestellung, findet sie nicht und hält sie für gelöscht. -->
			{#if gekappt}
				<p class="text-sm text-on-surface-variant">
					{bestellungen.length} von {gesamtImBlick} Bestellungen, die neuesten zuerst. Ältere stehen in
					den Bestellberichten.
				</p>
			{/if}
			<BestellHistorieTabelle {bestellungen} euro={formatEuro} {datum} onOeffnen={oeffne} />
		{/if}
	</div>
{/if}
