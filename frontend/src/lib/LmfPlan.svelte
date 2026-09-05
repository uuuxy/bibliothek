<!-- @component LmfPlan — der Planer: Der Plan ist eine REIHENFOLGE von Klassen, die der
     Server auf Schultage × Stunden gießt (Peter, 05.09.2026, am echten Plan der Schule:
     Abschlussklassen zuerst, dann jeder Schultag Stunde 1–6, die Reihenfolge läuft über
     die Tage weiter). Oben der Rahmen, darunter die Reihenfolge als die bekannte Tabelle
     mit Vorschau der Plätze, unten „Nicht im Plan". Feiertage rechnet der Server, freie
     Tage der Schule stehen am Plan, und eine Zeile kann ihren Platz fest bekommen (die
     Klasse mit dem Ausflug). Ein neuer Plan beginnt mit der Reihenfolge des Vorjahres. Gespeichert wird ausdrücklich — Rückgabe-Termine setzen
     Fristen. Menüpunkt System → „Schuljahreswechsel" (Peter, 05.09.2026 abends). -->
<script>
	import { onMount, untrack } from 'svelte';
	import PageShell from './components/layout/PageShell.svelte';
	import LadeFehler from './components/ui/LadeFehler.svelte';
	import LmfPlanKopf from './components/lmfplan/LmfPlanKopf.svelte';
	import LmfPlanRahmen from './components/lmfplan/LmfPlanRahmen.svelte';
	import LmfPlanFreieTage from './components/lmfplan/LmfPlanFreieTage.svelte';
	import LmfPlanReihenfolge from './components/lmfplan/LmfPlanReihenfolge.svelte';
	import LmfPlanVorrat from './components/lmfplan/LmfPlanVorrat.svelte';
	import { showToast } from '../inventur/lib/store.svelte.js';
	import * as dienst from './lmfplanDienst.js';

	let art = $state('rueckgabe');
	/** @type {import('./lmfplanDienst.js').PlanStand | null} */
	let stand = $state(null);
	/** @type {import('./lmfplanDienst.js').PlanEntwurf} */
	let entwurf = $state({
		erster_tag: '',
		startstunde: 1,
		stunden_je_tag: 6,
		freie_tage: [],
		zeilen: [],
		ausgelassen: []
	});
	/** @type {{ datum: string, stunde: number }[]} */
	let plaetze = $state([]);
	/** @type {import('./lmfplanDienst.js').Ausfall[]} */
	let ausfaelle = $state([]);
	let laedt = $state(true);
	let speichert = $state(false);
	// Gescheitertes Laden ist ein eigener Zustand, kein leerer Plan: Sonst stünde nach
	// einem Netzfehler „Noch kein Plan" da, der Planer böte die Regel-Reihenfolge an —
	// und ein Klick auf „Plan speichern" ersetzte den echten Plan des Schuljahres durch
	// diesen Entwurf und stellte die Fristen der Klassen auf den Stichtag zurück.
	// Dieselbe Klasse wie an den Einstellungen am 31.08.2026 (ui/LadeFehler.svelte).
	let ladeFehler = $state(false);

	async function lade() {
		laedt = true;
		try {
			stand = await dienst.ladeStand(art);
			entwurf = dienst.entwurfAus(stand);
			ladeFehler = false;
		} catch (e) {
			ladeFehler = true;
			showToast(`${e}`, 'error');
		} finally {
			laedt = false;
		}
	}

	// Vorschau: der Server rechnet die Plätze, sobald Rahmen oder Reihenfolge sich ändern
	// (entprellt). Liest den Entwurf, schreibt NUR plaetze — kein Effekt auf eigenen State.
	//
	// Verfolgt wird der Vorschau-Schlüssel (lmfplanDienst.vorschauSchluessel): Rahmen,
	// freie Tage, Anzahl der Zeilen und die festen Plätze je Zeile — nicht Klassen und
	// Vermerke, die ändern keinen Platz. Ein fester Platz dagegen verschiebt die Zeilen um
	// ihn herum, und wandert er beim Umsortieren in eine andere Zeile, gehört die
	// Verteilung neu gerechnet.
	//
	// `laufNr` ist die Sequenznummer wie im orderStore: Zwei schnelle Änderungen schicken
	// zwei Anfragen, und ohne Nummer könnte die ältere Antwort die jüngere überholen — die
	// Tabelle stünde dann mit den Daten eines Entwurfs da, den es nicht mehr gibt
	// (Rasterfrage 6, Frontend-Lesart).
	let laufNr = 0;
	$effect(() => {
		void dienst.vorschauSchluessel(entwurf);
		void entwurf.zeilen.length;
		const aktuelleArt = art;
		const timer = setTimeout(async () => {
			const meine = ++laufNr;
			try {
				const z = await dienst.rechneVorschau(
					aktuelleArt,
					untrack(() => JSON.parse(JSON.stringify(entwurf)))
				);
				if (meine !== laufNr) return; // eine jüngere Anfrage ist schon unterwegs oder da
				plaetze = z.plaetze.map((p) => ({ datum: p.datum, stunde: p.stunde }));
				ausfaelle = z.ausfaelle;
			} catch (e) {
				if (meine === laufNr) showToast(`${e}`, 'error');
			}
		}, 250);
		return () => clearTimeout(timer);
	});

	/** @param {string} k */
	function klasseRaus(k) {
		if (!entwurf.ausgelassen.some((x) => dienst.normKey(x) === dienst.normKey(k)))
			entwurf.ausgelassen = [...entwurf.ausgelassen, k].sort((a, b) =>
				a.localeCompare(b, 'de', { numeric: true })
			);
	}

	/** @param {string} k */
	function klasseHinein(k) {
		entwurf.ausgelassen = entwurf.ausgelassen.filter(
			(x) => dienst.normKey(x) !== dienst.normKey(k)
		);
		if (!entwurf.zeilen.some((z) => z.klassen.some((x) => dienst.normKey(x) === dienst.normKey(k))))
			entwurf.zeilen = [...entwurf.zeilen, { klassen: [k], vermerk: '' }];
	}

	const gueltig = $derived(
		Boolean(entwurf.erster_tag) &&
			entwurf.zeilen.every((z) => z.klassen.length > 0 || z.vermerk.trim() !== '') &&
			dienst.festePlaetzeVollstaendig(entwurf.zeilen)
	);

	async function speichern() {
		speichert = true;
		try {
			const erg = await dienst.speicherePlan(art, entwurf);
			showToast(erg.meldung, erg.ok ? 'success' : 'error');
			if (erg.ok) await lade();
		} finally {
			speichert = false;
		}
	}

	async function verwerfen() {
		if (!stand?.plan) return;
		if (!confirm(`Plan vom ${dienst.datumKurz(stand.plan.erster_tag)} verwerfen?`)) return;
		const erg = await dienst.verwerfePlan(art);
		showToast(erg.meldung, erg.ok ? 'success' : 'error');
		if (erg.ok) await lade();
	}

	async function pdf() {
		try {
			await dienst.ladePdf();
		} catch (e) {
			showToast(`${e}`, 'error');
		}
	}

	// Nicht `onMount(lade)`: Eine async-Funktion gibt eine Zusage zurück, und Svelte
	// nimmt den Rückgabewert von onMount als Aufräum-Funktion.
	onMount(() => {
		lade();
	});
</script>

<PageShell>
	<LmfPlanKopf
		{art}
		{stand}
		{laedt}
		{ladeFehler}
		{gueltig}
		{speichert}
		onart={(w) => {
			art = w;
			lade();
		}}
		onpdf={pdf}
		onverwerfen={verwerfen}
		onspeichern={speichern}
	/>

	{#if laedt}
		<div class="flex items-center justify-center py-12">
			<div
				class="h-8 w-8 animate-spin rounded-full border-2 border-surface-container-high border-t-primary"
			></div>
		</div>
	{:else if ladeFehler}
		<LadeFehler
			onerneut={lade}
			titel="Plan nicht geladen"
			text="Der gespeicherte Plan konnte nicht abgerufen werden. Der Planer bleibt geschlossen — sonst würde ein Klick auf „Plan speichern“ den echten Plan durch diesen Entwurf ersetzen und die Fristen der Klassen zurückstellen."
		/>
	{:else}
		<div class="mt-4 space-y-6">
			<LmfPlanRahmen
				bind:ersterTag={entwurf.erster_tag}
				bind:startstunde={entwurf.startstunde}
				bind:stundenJeTag={entwurf.stunden_je_tag}
			/>
			<LmfPlanFreieTage bind:tage={entwurf.freie_tage} {ausfaelle} />
			<LmfPlanReihenfolge bind:zeilen={entwurf.zeilen} {plaetze} onklasseraus={klasseRaus} />
			<LmfPlanVorrat klassen={entwurf.ausgelassen} onhinein={klasseHinein} />
		</div>
	{/if}
</PageShell>
