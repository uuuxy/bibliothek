<!-- @component LmfPlan — der Planer: eine REIHENFOLGE von Klassen, die der Server auf
     Schultage × Stunden gießt (Peter, 05.09.2026): Rahmen, Reihenfolge mit Vorschau der
     Plätze und, darüber, „Noch nicht im Plan". Gespeichert wird ein ENTWURF (Migration
     100), für Portal und Kollegiums-PDF unsichtbar; erst „Veröffentlichen" macht ihn
     gültig und setzt beim Büchertausch die Fristen (die Schulleitung nimmt ihn vorher
     ab). System → „Schuljahreswechsel".

     Aufbau seit 06.09.2026 nach Arbeitsablauf, nicht nach Datenmodell (Peter: „es steht
     oben was und unten was"): Kopf haftet oben, freie Tage als Chips, fehlende Klassen
     direkt über der Reihenfolge; einplanen setzt an den Platz (lmfplanZeilen.js). -->
<script>
	import { onMount, untrack } from 'svelte';
	import PageShell from './components/layout/PageShell.svelte';
	import LadeFehler from './components/ui/LadeFehler.svelte';
	import LmfPlanKopf from './components/lmfplan/LmfPlanKopf.svelte';
	import LmfPlanRahmen from './components/lmfplan/LmfPlanRahmen.svelte';
	import LmfPlanReihenfolge from './components/lmfplan/LmfPlanReihenfolge.svelte';
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
	/** Die gerade eingeplante Zeile — die Tabelle scrollt hin und hebt sie kurz hervor.
	 *  @type {{ index: number } | null} */
	let markiert = $state(null);
	let laedt = $state(true);
	let speichert = $state(false);
	// Gescheitertes Laden ist ein eigener Zustand, kein leerer Plan (ui/LadeFehler.svelte):
	// sonst ersetzte „Plan speichern" den echten Plan durch die Regel-Reihenfolge.
	let ladeFehler = $state(false);

	// `ladeNr` ist dieselbe Sequenznummer wie unten in der Vorschau (Rasterfrage 6) — der
	// Ladepfad hatte sie bis zum Rasterdurchgang am 06.09.2026 NICHT, zwei Zeilen neben
	// dem Kommentar, der sie erklärt. Beim Umschalten der Art laufen zwei GETs; kam die
	// ältere Antwort zuletzt, stand am Ende die Art des einen Plans über den Zeilen des
	// anderen — und „Plan speichern" schickte die Büchertausch-Reihenfolge an
	// PUT /api/lmf-plan/ausgabe, wo sie den (weiter veröffentlichten!) Ausgabe-Plan
	// überschrieb. Der Zustand heilte nicht von selbst: Die Vorschau rechnete die fremde
	// Reihenfolge anstandslos durch, die Tabelle sah stimmig aus.
	let ladeNr = 0;
	async function lade() {
		const meine = ++ladeNr;
		const meineArt = art;
		laedt = true;
		// Alles, was am vorigen Plan hing, geht zurück auf Anfang. Sonst stünden die Plätze
		// des anderen Plans in der Tabelle, bis die neue Vorschau kommt (250 ms + Rundlauf)
		// — und ein Klick auf eine Datumszelle nähme genau diesen fremden Platz als festen
		// Platz mit (LmfPlanPlatzZellen). `stand = null` sperrt zugleich die Kopf-Aktionen,
		// die sonst auf dem alten Plan arbeiten.
		stand = null;
		entwurf = dienst.leererEntwurf();
		plaetze = [];
		ausfaelle = [];
		beginn = null;
		markiert = null;
		try {
			const geladen = await dienst.ladeStand(meineArt);
			if (meine !== ladeNr) return; // eine jüngere Anfrage ist unterwegs oder schon da
			stand = geladen;
			entwurf = dienst.entwurfAus(geladen);
			ladeFehler = false;
		} catch (e) {
			if (meine !== ladeNr) return;
			ladeFehler = true;
			showToast(`${e}`, 'error');
		} finally {
			if (meine === ladeNr) laedt = false;
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
	const draussen = $derived(dienst.bewusstDraussen(stand));

	/** @param {string} k @param {number} [vor] */
	function klasseHinein(k, vor) {
		const erg = dienst.klasseHinein(entwurf, k, vor);
		entwurf = erg.entwurf;
		if (erg.index !== null) markiert = { index: erg.index };
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
		<div class="space-y-8">
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
				ausgelassen={entwurf.ausgelassen}
				{draussen}
				{markiert}
				onklasseraus={(k) => (entwurf = dienst.klasseRaus(entwurf, k))}
				onhinein={klasseHinein}
				ontausch={(i, alt, neu) => (entwurf = dienst.klasseTauschen(entwurf, i, alt, neu))}
			/>
		</div>
	{/if}
</PageShell>
