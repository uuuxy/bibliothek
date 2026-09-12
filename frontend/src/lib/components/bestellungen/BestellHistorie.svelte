<script>
	import { onMount } from 'svelte';
	import { apiGet } from '../../apiFetch.js';
	import BestellDetail from './BestellDetail.svelte';
	import BestellHistorieTabelle from './BestellHistorieTabelle.svelte';
	import BestellHistorieKopf from './BestellHistorieKopf.svelte';
	import { MITTEL, MITTEL_REIHENFOLGE } from './mittel.js';

	/** @type {any[]} */
	let bestellungen = $state([]);
	let loading = $state(true);
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
		const [liste, summen] = await Promise.all([
			apiGet('/api/bestellhistorie' + (mittel ? `?mittel=${mittel}` : '')),
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
	const topfLabel = (wert) =>
		wert && wert in MITTEL ? `${MITTEL[/** @type {any} */ (wert)].label}` : 'ohne Zuordnung';

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
		<BestellHistorieKopf
			{offeneBestaetigungen}
			{gesamtsumme}
			{gesamtExemplare}
			{aufteilung}
			{euro}
			{topfLabel}
			{mittelFilter}
			zeigeKennzahlen={bestellungen.length > 0}
			bind:mittel
			onFilterWechsel={ladeBestellungen}
		/>

		{#if loading}
			<div class="py-16 text-center text-slate-400 text-base animate-pulse">
				Lade Bestellhistorie…
			</div>
		{:else if bestellungen.length === 0}
			<div class="py-16 text-center text-slate-400 text-base">
				{#if mittel}
					Keine Bestellungen in diesem Topf.<br />
					<span class="text-sm">Andere Mittelherkunft wählen, um alle zu sehen.</span>
				{:else}
					Noch keine Bestellungen aufgegeben.<br />
					<span class="text-sm">Bestellungen werden hier automatisch gespeichert.</span>
				{/if}
			</div>
		{:else}
			<BestellHistorieTabelle {bestellungen} {euro} {datum} {kurzdatum} onOeffnen={oeffne} />

			<!-- Ehrlich sagen, dass die Liste nicht alles zeigt. Ohne den Satz sucht jemand eine
		     ältere Bestellung, findet sie nicht und hält sie für gelöscht. -->
			{#if gekappt}
				<p class="text-center text-xs text-slate-400">
					Neueste {bestellungen.length} von {gesamtImBlick} Bestellungen — ältere stehen im Bericht.
				</p>
			{/if}
		{/if}
	</div>
{/if}
