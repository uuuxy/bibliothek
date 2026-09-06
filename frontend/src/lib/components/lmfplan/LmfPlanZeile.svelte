<!-- @component LmfPlanZeile — eine Zeile der Reihenfolge: Nummer, dann Wochentag,
     Datum und Stunde (LmfPlanPlatzZellen: gerechnet und anklickbar, festgelegt als
     Felder), Klassen (LmfPlanKlassenZelle: Text oder Chips, anklickbar zum Tauschen),
     Besonderheiten, Aktionen. Ohne ersten Tag bleiben die gerechneten Spalten leer;
     der Abschnitt darüber sagt, was fehlt.

     Ziel und Marke: `ziel` zeichnet beim Ziehen die Einfügelinie über der Zeile (Apple
     HIG: „display an insertion point … only when the destination can accept a dragged
     item"); `markiert` hebt eine gerade eingeplante Zeile kurz hervor (M3: Auswahl =
     secondary-container), damit man sieht, wohin die Nachbar-Regel sie gesetzt hat. -->
<script>
	import Feld from '../ui/Feld.svelte';
	import LmfPlanKlassenZelle from './LmfPlanKlassenZelle.svelte';
	import LmfPlanPlatzZellen from './LmfPlanPlatzZellen.svelte';
	import LmfPlanZeileAktionen from './LmfPlanZeileAktionen.svelte';

	/** @type {{ zeile: import('../../lmfplanDienst.js').PlanZeile, i: number, anzahl: number, platz: { datum: string, stunde: number } | undefined, gezogen: boolean, ziel: boolean, markiert: boolean, marker: ReturnType<typeof import('../../lmfplanDienst.js').klassenMarker>, vorrat: string[], onziehstart: () => void, onziehueber: () => void, onablegen: (e: DragEvent) => void, onklasseraus: (klasse: string) => void, ontausch: (alt: string, neu: string) => void, onhoch: () => void, onrunter: () => void, onanfang: () => void, onende: () => void, onzusammen: () => void, ontrennen: () => void, oneinfuegen: () => void, onfest: () => void, onentfernen: () => void }} */
	let {
		zeile = $bindable(),
		i,
		anzahl,
		platz,
		gezogen,
		ziel,
		markiert,
		marker,
		vorrat,
		onziehstart,
		onziehueber,
		onablegen,
		onklasseraus,
		ontausch,
		onhoch,
		onrunter,
		onanfang,
		onende,
		onzusammen,
		ontrennen,
		oneinfuegen,
		onfest,
		onentfernen
	} = $props();

	const flaeche = $derived(
		markiert ? 'bg-secondary-container' : gezogen ? 'opacity-50' : 'hover:bg-surface-container-low'
	);
</script>

<tr
	id="lmf-zeile-{i}"
	draggable="true"
	ondragstart={onziehstart}
	ondragover={(e) => {
		e.preventDefault();
		onziehueber();
	}}
	ondrop={(e) => {
		e.preventDefault();
		onablegen(e);
	}}
	class="h-12 transition-colors {flaeche} {ziel ? '[&>td]:border-t-2 [&>td]:border-primary' : ''}"
>
	<td class="px-2 py-1 text-right tabular-nums text-on-surface-variant">{i + 1}</td>
	<LmfPlanPlatzZellen bind:zeile {i} {platz} {onfest} />
	<LmfPlanKlassenZelle klassen={zeile.klassen} {i} {vorrat} {marker} {onklasseraus} {ontausch} />
	<td class="px-4 py-1">
		<Feld
			id="lmf-zeile-vermerk-{i}"
			aria-label="Besonderheiten Zeile {i + 1}"
			bind:value={zeile.vermerk}
			placeholder={zeile.klassen.length === 0 ? 'Pflicht ohne Klasse' : ''}
			ungueltig={zeile.klassen.length === 0 && !zeile.vermerk.trim()}
			feld="w-full"
		/>
	</td>
	<td class="px-4 py-1 text-right whitespace-nowrap">
		<LmfPlanZeileAktionen
			nummer={i + 1}
			{anzahl}
			klassen={zeile.klassen.length}
			fest={Boolean(zeile.fest)}
			platzlos={!platz}
			{onhoch}
			{onrunter}
			{onanfang}
			{onende}
			{onzusammen}
			{ontrennen}
			{oneinfuegen}
			{onfest}
			onklasseraus={() => onklasseraus(zeile.klassen[0])}
			{onentfernen}
		/>
	</td>
</tr>
