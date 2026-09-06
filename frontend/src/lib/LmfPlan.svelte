<!-- @component LmfPlan — der Planer: eine REIHENFOLGE von Klassen, die der Server auf
     Schultage × Stunden gießt (Peter, 05.09.2026): Rahmen, Reihenfolge mit Vorschau der
     Plätze, „Nicht im Plan". Gespeichert wird ein ENTWURF (Migration 100), für Portal und
     Kollegiums-PDF unsichtbar; erst „Veröffentlichen" macht ihn gültig und setzt beim
     Büchertausch die Fristen (die Schulleitung nimmt ihn vorher ab). System → „Schuljahreswechsel". -->
<script>
	import { onMount, untrack } from 'svelte';
	import PageShell from './components/layout/PageShell.svelte';
	import LadeFehler from './components/ui/LadeFehler.svelte';
	import LmfPlanKopf from './components/lmfplan/LmfPlanKopf.svelte';
	import LmfPlanRahmen from './components/lmfplan/LmfPlanRahmen.svelte';
	import LmfPlanReihenfolge from './components/lmfplan/LmfPlanReihenfolge.svelte';
	import LmfPlanVorrat from './components/lmfplan/LmfPlanVorrat.svelte';
	import { showToast } from '../inventur/lib/store.svelte.js';
	import * as dienst from './lmfplanDienst.js';

	let art = $state('rueckgabe');
	/** @type {import('./lmfplanDienst.js').PlanStand | null} */
	let stand = $state(null);
	/** @type {import('./lmfplanDienst.js').PlanEntwurf} */
	let entwurf = $state(dienst.leererEntwurf());
	/** @type {{ datum: string, stunde: number }[]} */
	let plaetze = $state([]);
	/** @type {import('./lmfplanDienst.js').Ausfall[]} */
	let ausfaelle = $state([]);
	/** Der gerechnete Beginn des Büchertauschs (der Plan hängt am Ende) — vom Server.
	 *  @type {{ datum: string, stunde: number } | null} */
	let beginn = $state(null);
	let laedt = $state(true);
	let speichert = $state(false);
	// Gescheitertes Laden ist ein eigener Zustand, kein leerer Plan (ui/LadeFehler.svelte):
	// sonst ersetzte „Plan speichern" den echten Plan durch die Regel-Reihenfolge.
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

	// Vorschau: der Server rechnet die Plätze, sobald sich etwas ändert, wovon sie abhängen
	// (dienst.vorschauSchluessel, entprellt). Liest den Entwurf, schreibt NUR plaetze —
	// kein Effekt auf eigenen State. `laufNr` ist die Sequenznummer wie im orderStore:
	// Ohne sie könnte eine ältere Antwort die jüngere überholen (Rasterfrage 6).
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
				beginn = z.beginn;
			} catch (e) {
				if (meine === laufNr) showToast(`${e}`, 'error');
			}
		}, 250);
		return () => clearTimeout(timer);
	});

	const marker = $derived(dienst.klassenMarker(stand));

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
			entwurf.zeilen = [...entwurf.zeilen, { klassen: [k], vermerk: '', fest: null }];
	}

	const gueltig = $derived(
		dienst.ankerGesetzt(art, entwurf) &&
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

	// Veröffentlichen speichert den Entwurf zuerst — was die Schulleitung im PDF sah und
	// was das Kollegium gleich sieht, soll derselbe Stand sein.
	async function veroeffentlichen() {
		const fristen = art === 'rueckgabe' ? ', und die Termine werden die Fristen der Klassen' : '';
		if (!confirm(`Plan veröffentlichen? Das Kollegium sieht ihn dann im Portal${fristen}.`)) return;
		speichert = true;
		try {
			const gespeichert = await dienst.speicherePlan(art, entwurf);
			if (!gespeichert.ok) {
				showToast(gespeichert.meldung, 'error');
				return;
			}
			const erg = await dienst.veroeffentlichePlan(art);
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

	// Mit Entwurf: das PDF geht zur Abnahme an die Schulleitung.
	const pdf = () => dienst.ladePdf(false, true).catch((e) => showToast(`${e}`, 'error'));

	// Nicht `onMount(lade)`: Svelte nähme die zurückgegebene Zusage als Aufräum-Funktion.
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
		onveroeffentlichen={veroeffentlichen}
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
		<div class="mt-6 space-y-8">
			<LmfPlanRahmen
				{art}
				bind:entwurf
				{ausfaelle}
				{beginn}
				sommerferien={stand?.sommerferien ?? null}
			/>
			<LmfPlanReihenfolge
				bind:zeilen={entwurf.zeilen}
				{plaetze}
				{marker}
				bereit={dienst.ankerGesetzt(art, entwurf)}
				onklasseraus={klasseRaus}
			/>
			<LmfPlanVorrat klassen={entwurf.ausgelassen} {marker} onhinein={klasseHinein} />
		</div>
	{/if}
</PageShell>
