<!-- @component LmfPlanReihenfolge — Abschnitt „Reihenfolge": der Plan als Tabelle in
     der Form, die das Kollegium kennt (Wochentag, Datum, Stunde, Klassen,
     Besonderheiten), nur dass sie hier bearbeitet wird: Zeilen ziehen oder mit den
     Pfeilen schieben, zwei Zeilen zu einer Stunde zusammenlegen („10R1/10R2"), eine
     Zeile ohne Klasse davor einfügen („Bücher setzen"), festlegen, Klasse aus dem Plan
     nehmen. Die Umformungen rechnet lmfplanZeilen.js, die Zeile selbst ist
     LmfPlanZeile, ihre Aktionen LmfPlanZeileAktionen. Seit dem 06.09.2026 sind die
     Zellen selbst der kurze Weg: Klick auf Wochentag, Datum oder Stunde legt die Zeile
     fest, Klick auf die Klasse tauscht sie gegen eine aus „Noch nicht im Plan".

     Über der Tabelle liegt seit dem 06.09.2026 LmfPlanVorrat („Noch nicht im Plan"):
     Ein Klick plant die Klasse an ihren Platz (Nachbar-Regel), ein Chip lässt sich auf
     eine Zeile ziehen und landet davor; die neue Zeile wird angescrollt und leuchtet
     kurz. Nach M3 Lists ohne Trennlinie je Zeile („Limit dividers … only when a
     stronger visual separation is necessary"): Zeilenhöhe 48 px und die Hover-Fläche
     trennen, die Linie steht nur unter dem Kopf. -->
<script>
	import Button from '../ui/Button.svelte';
	import LmfPlanVorrat from './LmfPlanVorrat.svelte';
	import LmfPlanZeile from './LmfPlanZeile.svelte';
	import * as op from '../../lmfplanZeilen.js';

	/** @type {{ zeilen: import('../../lmfplanDienst.js').PlanZeile[], plaetze: { datum: string, stunde: number }[], marker: ReturnType<typeof import('../../lmfplanDienst.js').klassenMarker>, bereit: boolean, ausgelassen: string[], draussen: (klasse: string) => boolean, markiert: { index: number } | null, onklasseraus: (klasse: string) => void, onhinein: (klasse: string, vor?: number) => void, ontausch: (i: number, alt: string, neu: string) => void }} */
	let {
		zeilen = $bindable(),
		plaetze,
		marker,
		bereit,
		ausgelassen,
		draussen,
		markiert,
		onklasseraus,
		onhinein,
		ontausch
	} = $props();

	/** @type {number | null} */
	let gezogen = $state(null);
	/** @type {number | null} */
	let ziel = $state(null);
	/** @type {number | null} */
	let leuchtet = $state(null);

	// Eine gerade eingeplante Zeile: anscrollen und zwei Sekunden hervorheben. Liest nur
	// die Prop, schreibt nur `leuchtet` — kein Effekt auf eigenen State.
	$effect(() => {
		if (!markiert) return;
		leuchtet = markiert.index;
		document
			.getElementById(`lmf-zeile-${markiert.index}`)
			?.scrollIntoView({ block: 'center', behavior: 'smooth' });
		const timer = setTimeout(() => (leuchtet = null), 2000);
		return () => clearTimeout(timer);
	});

	// Klassen im Plan ohne Schüler (Vorjahr, vor dem Import getippt) — der Satz oben
	// zählt sie, damit man nach dem LUSD-Import sieht, was übrig geblieben ist.
	const ohneSchueler = $derived(
		[...new Set(zeilen.flatMap((z) => z.klassen))].filter((k) => marker.ohneSchueler(k))
	);

	/** Ablegen auf Zeile i: eine gezogene Zeile wandert dorthin, ein gezogener Chip
	 *  („Noch nicht im Plan") wird davor eingeplant.
	 *  @param {DragEvent} e @param {number} i */
	function ablegen(e, i) {
		const klasse = e.dataTransfer?.getData('text/lmf-klasse');
		if (gezogen !== null) zeilen = op.verschiebe(zeilen, gezogen, i);
		else if (klasse) onhinein(klasse, i);
		gezogen = null;
		ziel = null;
	}

	/** @param {number} i */
	function entfernen(i) {
		for (const k of zeilen[i].klassen) onklasseraus(k);
		zeilen = op.entfernen(zeilen, i);
	}

	/** @param {number} i @param {string} k */
	function klasseRaus(i, k) {
		onklasseraus(k);
		zeilen = op.klasseRaus(zeilen, i, k);
	}
</script>

<section aria-labelledby="lmf-reihenfolge-titel">
	<h2 id="lmf-reihenfolge-titel" class="text-title-medium font-medium text-on-surface">
		Reihenfolge
	</h2>
	<!-- Kein Bedienungssatz mehr (06.09.2026, Peter: Erklärungstexte kosten Zeilen).
	     Der Satz erscheint nur, wenn er etwas zu sagen hat: der fehlende erste Tag oder
	     Klassen ohne Schüler. -->
	{#if !bereit || ohneSchueler.length > 0}
		<p class="mt-1 max-w-3xl text-sm text-on-surface-variant" data-testid="lmf-reihenfolge-hinweis">
			{#if !bereit}
				Ersten Tag wählen — dann rechnet der Plan Wochentag, Datum und Stunde jeder Zeile.
			{/if}
			{#if ohneSchueler.length > 0}
				<span data-testid="lmf-ohne-schueler">
					{ohneSchueler.length === 1 ? 'Eine Klasse' : `${ohneSchueler.length} Klassen`} im Plan
					{ohneSchueler.length === 1 ? 'hat' : 'haben'} noch keine Schüler ({ohneSchueler.join(
						', '
					)}) — sie kommen mit dem LUSD-Import oder gehören aus dem Plan.
				</span>
			{/if}
		</p>
	{/if}
	<LmfPlanVorrat klassen={ausgelassen} {draussen} {marker} onhinein={(k) => onhinein(k)} />
	<div class="mt-4 overflow-x-auto" ondragleave={() => (ziel = null)} role="presentation">
		<table class="w-full border-collapse text-left text-sm" data-testid="lmf-reihenfolge">
			<thead>
				<!-- Spaltenbreiten (06.09.2026): Die gerechneten Spalten und die Aktionen sind so
				     schmal wie ihr Inhalt, Klassen bekommen festen Platz für zwei Chips, und die
				     Besonderheiten füllen den Rest — vorher lagen 300 px Leere rechts vom Feld. -->
				<tr class="border-b border-outline-variant text-on-surface-variant">
					<th class="w-10 px-2 py-2 text-right font-medium">#</th>
					<th class="w-px px-4 py-2 font-medium whitespace-nowrap">Wochentag</th>
					<th class="w-px px-4 py-2 font-medium whitespace-nowrap">Datum</th>
					<th class="w-px px-4 py-2 font-medium whitespace-nowrap">Stunde</th>
					<th class="w-48 px-4 py-2 font-medium">Klassen</th>
					<th class="px-4 py-2 font-medium">Besonderheiten</th>
					<th class="w-px px-4 py-2 text-right font-medium whitespace-nowrap">Aktionen</th>
				</tr>
			</thead>
			<tbody>
				{#each zeilen, i (i)}
					<LmfPlanZeile
						bind:zeile={zeilen[i]}
						{i}
						anzahl={zeilen.length}
						platz={plaetze[i]}
						{marker}
						vorrat={ausgelassen}
						gezogen={gezogen === i}
						ziel={ziel === i}
						markiert={leuchtet === i}
						onziehstart={() => (gezogen = i)}
						onziehueber={() => (ziel = i)}
						onablegen={(e) => ablegen(e, i)}
						onklasseraus={(k) => klasseRaus(i, k)}
						ontausch={(alt, neu) => ontausch(i, alt, neu)}
						onhoch={() => (zeilen = op.verschiebe(zeilen, i, i - 1))}
						onrunter={() => (zeilen = op.verschiebe(zeilen, i, i + 1))}
						onanfang={() => (zeilen = op.verschiebe(zeilen, i, 0))}
						onende={() => (zeilen = op.verschiebe(zeilen, i, zeilen.length - 1))}
						onzusammen={() => (zeilen = op.zusammenlegen(zeilen, i))}
						ontrennen={() => (zeilen = op.trennen(zeilen, i))}
						oneinfuegen={() => (zeilen = op.einfuegen(zeilen, i))}
						onfest={() => (zeilen = op.festWechseln(zeilen, i, plaetze[i]))}
						onentfernen={() => entfernen(i)}
					/>
				{/each}
			</tbody>
		</table>
		{#if zeilen.length === 0}
			<p class="px-4 py-6 text-sm text-on-surface-variant">
				Noch keine Zeile — eine Klasse aus „Noch nicht im Plan" einplanen.
			</p>
		{/if}
		<div class="px-2 pt-2">
			<Button
				variant="ghost"
				size="sm"
				onclick={() => (zeilen = op.einfuegen(zeilen, zeilen.length))}
			>
				Zeile ohne Klasse anhängen
			</Button>
		</div>
	</div>
</section>
