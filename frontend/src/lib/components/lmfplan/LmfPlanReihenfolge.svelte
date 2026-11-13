<!-- @component LmfPlanReihenfolge — Abschnitt „Reihenfolge": der Plan als Tabelle in
     der Form, die das Kollegium kennt (Wochentag, Datum, Stunde, Klassen,
     Besonderheiten), nur dass sie hier bearbeitet wird: Zeilen ziehen oder über ihr Menü
     verschieben, zwei Zeilen zu einer Stunde zusammenlegen („10R1/10R2"), eine
     Zeile ohne Klasse davor einfügen („Bücher setzen"), festlegen, Klasse aus dem Plan
     nehmen. Die Umformungen rechnet lmfplanZeilen.js, die Zeile selbst ist
     LmfPlanZeile, ihre Aktionen LmfPlanZeileAktionen. Die Zellen sind selbst der kurze
     Weg: Klick auf Wochentag, Datum oder Stunde legt die Zeile fest, Klick auf die Klasse
     tauscht sie gegen eine aus „Noch nicht im Plan".

     Über der Tabelle liegt LmfPlanVorrat („Noch nicht im Plan"):
     Ein Klick plant die Klasse an ihren Platz (Nachbar-Regel), ein Chip lässt sich auf
     eine Zeile ziehen und landet davor; die neue Zeile wird angescrollt und leuchtet
     kurz. Nach M3 Lists steht eine Trennlinie nur zwischen zwei Tagen („Limit dividers …
     only when a stronger visual separation is necessary"); innerhalb eines Tages trennen
     Zeilenhöhe und Hover-Fläche. -->
<script>
	import Button from '../ui/Button.svelte';
	import Tabelle from '../ui/Tabelle.svelte';
	import LmfPlanVorrat from './LmfPlanVorrat.svelte';
	import LmfPlanZeile from './LmfPlanZeile.svelte';
	import * as op from './lmfplanZeilen.js';

	/** @type {{ zeilen: import('./lmfplanDienst.js').PlanZeile[], plaetze: { datum: string, stunde: number }[], marker: ReturnType<typeof import('./lmfplanDienst.js').klassenMarker>, bereit: boolean, ausgelassen: string[], draussen: (klasse: string) => boolean, markiert: { index: number } | null, onklasseraus: (klasse: string) => void, onhinein: (klasse: string, vor?: number) => void, ontausch: (i: number, alt: string, neu: string) => void }} */
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

	// Erst die Zeile ändern, dann melden: Der Planer merkt eine Klasse nur als ausgelassen,
	// wenn sie in keiner Zeile mehr steht, und liest dafür die gebundenen Zeilen.
	/** @param {number} i */
	function entfernen(i) {
		const klassen = zeilen[i].klassen;
		zeilen = op.entfernen(zeilen, i);
		for (const k of klassen) onklasseraus(k);
	}

	/** @param {number} i @param {string} k */
	function klasseRaus(i, k) {
		zeilen = op.klasseRaus(zeilen, i, k);
		onklasseraus(k);
	}
</script>

<section aria-labelledby="lmf-reihenfolge-titel">
	<h2 id="lmf-reihenfolge-titel" class="text-base font-medium text-on-surface">Reihenfolge</h2>
	<!-- Kein Bedienungssatz, denn Erklärungstexte kosten Zeilen. Der Satz erscheint nur, wenn
	     der erste Tag fehlt; eine Klasse ohne Schüler zeigt ihr Chip in der Zeile. -->
	{#if !bereit}
		<p class="mt-1 max-w-3xl text-sm text-on-surface-variant" data-testid="lmf-reihenfolge-hinweis">
			Ersten Tag wählen — dann rechnet der Plan Wochentag, Datum und Stunde jeder Zeile.
		</p>
	{/if}
	<LmfPlanVorrat klassen={ausgelassen} {draussen} {marker} onhinein={(k) => onhinein(k)} />
	<div class="mt-4 overflow-x-auto" ondragleave={() => (ziel = null)} role="presentation">
		<Tabelle beschriftung="Reihenfolge der Klassen" data-testid="lmf-reihenfolge">
			<thead>
				<!-- Spaltenbreiten: Die gerechneten Spalten und die Aktionen sind so schmal wie ihr
				     Inhalt, Klassen bekommen festen Platz für zwei Chips, und die Besonderheiten
				     füllen den Rest, damit rechts vom Feld keine Leere liegt. -->
				<tr>
					<th class="w-10 text-right">#</th>
					<th class="w-px whitespace-nowrap">Wochentag</th>
					<th class="w-px whitespace-nowrap">Datum</th>
					<th class="w-px whitespace-nowrap">Stunde</th>
					<th class="w-48">Klassen</th>
					<th>Besonderheiten</th>
					<th class="w-px text-right whitespace-nowrap">Aktionen</th>
				</tr>
			</thead>
			<tbody>
				{#each zeilen, i (i)}
					<LmfPlanZeile
						bind:zeile={zeilen[i]}
						{i}
						platz={plaetze[i]}
						{marker}
						vorrat={ausgelassen}
						gezogen={gezogen === i}
						ziel={ziel === i}
						markiert={leuchtet === i}
						tagGehtWeiter={op.tagGehtWeiter(plaetze, i)}
						onziehstart={() => (gezogen = i)}
						onziehueber={() => (ziel = i)}
						onablegen={(e) => ablegen(e, i)}
						onklasseraus={(k) => klasseRaus(i, k)}
						ontausch={(alt, neu) => ontausch(i, alt, neu)}
						onzusammen={() => (zeilen = op.zusammenlegen(zeilen, i))}
						ontrennen={() => (zeilen = op.trennen(zeilen, i))}
						oneinfuegen={() => (zeilen = op.einfuegen(zeilen, i))}
						onfest={() => (zeilen = op.festWechseln(zeilen, i, plaetze[i]))}
						onentfernen={() => entfernen(i)}
					/>
				{/each}
			</tbody>
		</Tabelle>
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
