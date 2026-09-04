<!-- @component EtikettenNachdruck — findet Exemplare, deren Barcode-Etikett nie gedruckt
     wurde, und übergibt sie an den Etikettendruck.

     Der Anlass: Eine Lieferung kann im System freigegeben sein, ohne dass die Etiketten
     je aus dem Drucker kamen — der Hinweis nach dem Wareneingang wird weggeklickt, und
     danach führte kein Weg mehr zu genau diesen Exemplaren zurück. Man hätte jeden Titel
     einzeln suchen müssen, ohne zu wissen, welche es überhaupt sind.

     Kein zweiter Druckweg: Die Auswahl geht in dieselbe printQueue, die auch der
     Wareneingang benutzt, und wird vom Etikettendruck nebenan gesetzt.

     Aufbau seit dem M3-Durchgang am 04.09.2026: EINE Aktionszeile unter der Suche, in der
     der gefüllte Knopf immer die Aktion der gewählten Stufe trägt. Vorher gab es zwei
     Orte für Aktionen — den gefüllten Knopf und einen Kasten, der bei Auswahl aufsprang.
     In „Erledigt" war der gefüllte Knopf dabei grundsätzlich unerreichbar: Er hing an
     `gewaehltOffen`, und in dieser Stufe gibt es davon keine. Gemessen mit 300 angehakten
     Zeilen stand der auffälligste Knopf der Seite auf „Nichts ausgewählt". -->
<script>
	import { onMount } from 'svelte';
	import { apiGet, apiPost } from '../../apiFetch.js';
	import { printQueue } from '../../stores/printQueue.svelte.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import { Printer, Eraser, Undo2 } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import Segmente from '../ui/Segmente.svelte';
	import Suchpille from '../ui/Suchpille.svelte';
	import EtikettenListe from './EtikettenListe.svelte';
	import AltbestandDialog from './AltbestandDialog.svelte';

	/** @type {{ onUebergeben?: () => void }} */
	let { onUebergeben } = $props();

	/** @type {{ barcode_id: string, titel: string, autor: string, erworben_am: string, etikett_gedruckt: boolean }[]} */
	let offen = $state.raw([]);
	let laedt = $state(true);
	let suche = $state('');
	let arbeitet = $state(false);
	let altbestandOffen = $state(false);

	/**
	 * Wie viele Exemplare der Filter INSGESAMT trifft — die Liste ist bei 300 gedeckelt
	 * (etikettenOffenLimit). Bis zum 04.09.2026 sagte das niemand: Am Reiter stand
	 * „30674", darunter lagen 300 Zeilen, und nichts verband die beiden Zahlen.
	 */
	let gesamt = $state(0);

	/**
	 * Welche Exemplare die Liste zeigt: 'offen' (Vorgabe), 'erledigt' oder 'alle'.
	 *
	 * Erledigte sichtbar zu machen ist der WEG ZURÜCK. Bisher setzten alle drei Wege das
	 * Kennzeichen nur in eine Richtung — Stapeldruck, Einzeldruck, Altbestand-Stichtag.
	 * Blieb der Bogen im Drucker stecken oder war der Stichtag zu weit gefasst, galten die
	 * Exemplare als erledigt, ohne dass ein Etikett existierte, und verschwanden dauerhaft
	 * aus dieser Liste. Wer sie nicht sehen kann, kann sie auch nicht zurückholen.
	 */
	let status = $state(/** @type {'offen' | 'erledigt' | 'alle'} */ ('offen'));
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let sucheTimer;

	/** Barcodes der angehakten Zeilen. */
	let gewaehlt = $state(/** @type {string[]} */ ([]));

	/**
	 * Laufende Nummer der Abfrage. Zwei Ladevorgänge können sich überholen — Tippen in der
	 * Suche und ein Klick auf eine andere Stufe stoßen beide `laden()` an, und die
	 * langsamere Antwort würde sonst die schnellere überschreiben.
	 */
	let lauf = 0;

	async function laden() {
		const meiner = ++lauf;
		laedt = true;
		try {
			const q = suche.trim();
			// Von Hand zusammengesetzt statt per URLSearchParams: Das ist hier ein
			// Wegwerf-String, und die Svelte-Lint-Regel verlangt für die eingebaute Klasse
			// sonst die reaktive Variante — Aufwand ohne Gegenwert.
			const teile = [];
			if (q) teile.push(`q=${encodeURIComponent(q)}`);
			if (status !== 'offen') teile.push(`status=${status}`);
			const query = teile.length > 0 ? `?${teile.join('&')}` : '';

			// Beide Wege tragen dieselben Filter — sonst nennt die Fußzeile eine Zahl aus
			// einer anderen Menge als die Zeilen darüber. Der Zähler darf scheitern, ohne
			// die Liste mitzureißen: Er ist die Beschriftung, nicht der Inhalt.
			const zaehlerP = apiGet(`/api/exemplare/etiketten-offen/anzahl${query}`).catch(() => null);
			const liste = (await apiGet(`/api/exemplare/etiketten-offen${query}`)) || [];
			const zaehler = await zaehlerP;
			if (meiner !== lauf) return;

			offen = liste;
			gesamt = zaehler?.anzahl ?? liste.length;
			// Auswahl auf das beschränken, was noch in der Liste steht — sonst übergäbe ein
			// Klick auf "Drucken" Exemplare, die der Benutzer gar nicht mehr sieht.
			const sichtbar = new Set(offen.map((e) => e.barcode_id));
			gewaehlt = gewaehlt.filter((b) => sichtbar.has(b));
		} catch (err) {
			if (meiner !== lauf) return;
			console.error('Offene Etiketten konnten nicht geladen werden', err);
			toastStore.addToast('Liste konnte nicht geladen werden.', 'error');
		} finally {
			if (meiner === lauf) laedt = false;
		}
	}

	onMount(() => {
		// Ein Verweis aus der Bestellhistorie bringt den Titel mit. Sofort zurücksetzen,
		// sonst klebt der Filter am nächsten Aufruf der Liste, und der Benutzer sucht den
		// Grund für eine halbleere Ansicht.
		if (uiStore.requestedEtikettenFilter) {
			suche = uiStore.requestedEtikettenFilter;
			uiStore.requestedEtikettenFilter = null;
		}
		laden();
	});

	function sucheAngestossen() {
		clearTimeout(sucheTimer);
		sucheTimer = setTimeout(laden, 300);
	}

	/** @param {'offen'|'erledigt'|'alle'} neu */
	function stufeWechseln(neu) {
		status = neu;
		gewaehlt = [];
		laden();
	}

	/** @param {string} barcode */
	function umschalten(barcode) {
		gewaehlt = gewaehlt.includes(barcode)
			? gewaehlt.filter((b) => b !== barcode)
			: [...gewaehlt, barcode];
	}

	function alleUmschalten() {
		gewaehlt = gewaehlt.length === offen.length ? [] : offen.map((e) => e.barcode_id);
	}

	/** Die angehakten Zeilen, getrennt nach dem, was mit ihnen möglich ist. */
	let gewaehltOffen = $derived(
		offen.filter((e) => gewaehlt.includes(e.barcode_id) && !e.etikett_gedruckt)
	);
	let gewaehltErledigt = $derived(
		offen.filter((e) => gewaehlt.includes(e.barcode_id) && e.etikett_gedruckt)
	);

	function uebergeben() {
		const auswahl = offen.filter((e) => gewaehlt.includes(e.barcode_id));
		if (auswahl.length === 0) return;
		printQueue.copies = auswahl.map((e) => ({
			barcode_id: e.barcode_id,
			titel: e.titel,
			autor: e.autor
		}));
		toastStore.addToast(
			`${auswahl.length} ${auswahl.length === 1 ? 'Etikett' : 'Etiketten'} im Druck übernommen.`,
			'success'
		);
		onUebergeben?.();
	}

	/**
	 * Von Hand als gedruckt vermerken — OHNE zu drucken.
	 *
	 * Für den Fall, dass die Etiketten anderswo entstanden sind: aus dem Altsystem, von
	 * Hand geschrieben, oder weil der Lieferant beklebt geliefert hat und die Einstellung
	 * beim Bestellen noch nicht gesetzt war.
	 */
	async function alsGedrucktMarkieren() {
		await vermerken(
			'/api/exemplare/etiketten-gedruckt',
			gewaehltOffen.map((e) => e.barcode_id),
			(n) => `${n} als erledigt vermerkt.`
		);
	}

	/**
	 * Den Vermerk zurücknehmen — der eigentliche Notfallweg.
	 *
	 * Der Druck wird gegengebucht, sobald das PDF erzeugt ist; ob das Etikett wirklich aus
	 * dem Gerät kam, weiss das Programm nicht. Nach einem Papierstau stehen die Exemplare
	 * also als erledigt da, ohne Etikett am Buch. Dasselbe nach einem zu weit gefassten
	 * Stichtag beim Altbestand-Aufräumen.
	 */
	async function wiederOeffnen() {
		await vermerken(
			'/api/exemplare/etiketten-zuruecksetzen',
			gewaehltErledigt.map((e) => e.barcode_id),
			(n) => `${n} wieder als offen vermerkt.`
		);
	}

	/**
	 * @param {string} pfad
	 * @param {string[]} barcodes
	 * @param {(anzahl: number) => string} meldung
	 */
	async function vermerken(pfad, barcodes, meldung) {
		if (barcodes.length === 0 || arbeitet) return;
		arbeitet = true;
		try {
			const daten = await apiPost(pfad, { barcode_ids: barcodes });
			toastStore.addToast(meldung(daten?.markiert ?? daten?.zurueckgesetzt ?? 0), 'success');
			await laden();
		} catch {
			toastStore.addToast('Vermerken nicht möglich.', 'error');
		} finally {
			arbeitet = false;
		}
	}

	/**
	 * Die eine Aktion, die der gefüllte Knopf trägt. Sie folgt der Stufe: In „Erledigt"
	 * gibt es nichts zu drucken, dort ist das Zurückholen der Hauptweg.
	 */
	let hauptaktion = $derived(
		status === 'erledigt'
			? {
					anzahl: gewaehltErledigt.length,
					text: 'wieder als offen vermerken',
					leer: 'Wieder als offen vermerken',
					tun: wiederOeffnen
				}
			: {
					anzahl: gewaehlt.length,
					text: 'an den Druck übergeben',
					leer: 'An den Druck übergeben',
					tun: uebergeben
				}
	);

	/** @param {number} n */
	const zahl = (n) => n.toLocaleString('de-DE');
</script>

<div class="no-print animate-fade-in w-full">
	<div class="flex flex-col gap-4 border-b border-outline-variant pb-4">
		<Suchpille
			id="etiketten-suchfeld"
			bind:wert={suche}
			oninput={sucheAngestossen}
			platzhalter="Titel oder Barcode eingeben …"
			etikett="Exemplare filtern"
		/>

		<div class="flex flex-wrap items-center gap-3">
			<!-- Vorgabe „Offen": Die Ansicht heisst „Fehlende Etiketten" und soll ohne Zutun
			     genau das zeigen; die anderen Stufen sind Notfall-Werkzeug. -->
			<Segmente
				etikett="Welche Exemplare"
				wert={status}
				onwahl={(w) => stufeWechseln(/** @type {'offen'|'erledigt'|'alle'} */ (w))}
				optionen={[
					{ wert: 'offen', text: 'Offen' },
					{ wert: 'erledigt', text: 'Erledigt' },
					{ wert: 'alle', text: 'Alle' }
				]}
			/>

			<Button
				size="lg"
				onclick={hauptaktion.tun}
				disabled={hauptaktion.anzahl === 0 || arbeitet}
				class="px-5"
			>
				{#if status === 'erledigt'}
					<Undo2 class="h-4 w-4" aria-hidden="true" />
				{:else}
					<Printer class="h-4 w-4" aria-hidden="true" />
				{/if}
				{hauptaktion.anzahl === 0
					? hauptaktion.leer
					: `${zahl(hauptaktion.anzahl)} ${hauptaktion.text}`}
			</Button>

			<!-- Die Nebenwege stehen in derselben Zeile, nicht in einem Kasten, der bei
			     Auswahl aufspringt: M3 kennt dafür den Text-Button neben dem gefüllten. -->
			{#if status !== 'erledigt' && gewaehltOffen.length > 0}
				<Button
					variant="ghost"
					size="lg"
					disabled={arbeitet}
					onclick={alsGedrucktMarkieren}
					data-tip="Etikett ist schon am Buch — z. B. vom Lieferanten oder aus dem Altsystem"
				>
					<Eraser class="h-4 w-4" aria-hidden="true" />
					{zahl(gewaehltOffen.length)} als erledigt vermerken
				</Button>
			{/if}
			{#if status !== 'erledigt' && gewaehltErledigt.length > 0}
				<Button
					variant="ghost"
					size="lg"
					disabled={arbeitet}
					onclick={wiederOeffnen}
					data-tip="Etikett fehlt doch — z. B. nach Papierstau oder zu weitem Stichtag"
				>
					<Undo2 class="h-4 w-4" aria-hidden="true" />
					{zahl(gewaehltErledigt.length)} wieder als offen vermerken
				</Button>
			{/if}
			{#if status === 'erledigt' && gewaehltErledigt.length > 0}
				<Button variant="ghost" size="lg" disabled={arbeitet} onclick={uebergeben}>
					<Printer class="h-4 w-4" aria-hidden="true" />
					{zahl(gewaehltErledigt.length)} noch einmal drucken
				</Button>
			{/if}

			<!-- Bei einem Altbestand ohne Etikett-Vermerk ist DAS die Aufgabe der Seite.
			     Deshalb steht sie oben und nicht mehr am Fuß hinter 300 Zeilen. -->
			<Button variant="ghost" size="lg" class="ml-auto" onclick={() => (altbestandOffen = true)}>
				Altbestand aufräumen
			</Button>
		</div>
	</div>

	{#if laedt}
		<p class="animate-pulse py-16 text-center text-on-surface-variant">Lade Exemplare…</p>
	{:else if offen.length === 0}
		<div class="py-16 text-center text-on-surface-variant">
			{#if suche.trim() && status === 'erledigt'}
				Kein erledigtes Exemplar passt zu „{suche.trim()}".
			{:else if suche.trim() && status === 'alle'}
				Kein Exemplar passt zu „{suche.trim()}".
			{:else if suche.trim()}
				Kein Exemplar ohne Etikett passt zu „{suche.trim()}".
			{:else if status === 'erledigt'}
				Noch kein Exemplar als erledigt vermerkt.
			{:else if status === 'alle'}
				Keine Exemplare im Bestand.
			{:else}
				Für alle Exemplare wurden Etiketten gedruckt.
			{/if}
		</div>
	{:else}
		<!-- ÜBER der Liste, nicht darunter: Die Zeile sagt, dass hier nur ein Ausschnitt
		     steht — und unter 300 Zeilen liest sie niemand. Beim ersten Anlauf stand sie am
		     Fuß, und im Screenshot war von ihr nichts zu sehen. -->
		<p class="py-3 text-sm text-on-surface-variant">
			{#if gesamt > offen.length}
				<strong class="font-medium text-on-surface">{zahl(offen.length)} von {zahl(gesamt)}</strong>
				— die neuesten zuerst. Suche nach Titel oder Barcode, um an die übrigen zu kommen.
			{:else}
				{zahl(offen.length)}
				{offen.length === 1 ? 'Exemplar' : 'Exemplare'}, neueste zuerst.
			{/if}
		</p>
		<EtikettenListe
			zeilen={offen}
			{gewaehlt}
			{status}
			onumschalten={umschalten}
			onalleUmschalten={alleUmschalten}
		/>
	{/if}

	<AltbestandDialog
		open={altbestandOffen}
		onclose={() => (altbestandOffen = false)}
		onfertig={laden}
	/>
</div>
