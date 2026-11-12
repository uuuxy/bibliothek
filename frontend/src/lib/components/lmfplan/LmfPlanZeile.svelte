<!-- @component LmfPlanZeile — eine Zeile der Reihenfolge: Nummer mit Griff, dann Wochentag,
     Datum und Stunde (LmfPlanPlatzZellen: gerechnet und anklickbar, festgelegt als
     Felder), Klassen (LmfPlanKlassenZelle: Text oder Chips, anklickbar zum Tauschen),
     Besonderheiten (LmfPlanVermerkZelle: Text, anklickbar zum Schreiben), Aktionen im Menü.
     Ohne ersten Tag bleiben die gerechneten Spalten leer; der Abschnitt darüber sagt, was
     fehlt. Folgt derselbe Tag, fällt die Linie unter der Zeile weg (`tagGehtWeiter`).

     Ziel und Marke: `ziel` zeichnet beim Ziehen die Einfügelinie über der Zeile (Apple
     HIG: „display an insertion point … only when the destination can accept a dragged
     item"); `markiert` hebt eine gerade eingeplante Zeile kurz hervor (M3: Auswahl =
     secondary-container), damit man sieht, wohin die Nachbar-Regel sie gesetzt hat. -->
<script>
	import { GripVertical } from '@lucide/svelte';
	import LmfPlanKlassenZelle from './LmfPlanKlassenZelle.svelte';
	import LmfPlanPlatzZellen from './LmfPlanPlatzZellen.svelte';
	import LmfPlanVermerkZelle from './LmfPlanVermerkZelle.svelte';
	import LmfPlanZeileAktionen from './LmfPlanZeileAktionen.svelte';

	/** @type {{ zeile: import('./lmfplanDienst.js').PlanZeile, i: number, platz: { datum: string, stunde: number } | undefined, gezogen: boolean, ziel: boolean, markiert: boolean, tagGehtWeiter: boolean, marker: ReturnType<typeof import('./lmfplanDienst.js').klassenMarker>, vorrat: string[], onziehstart: () => void, onziehueber: () => void, onablegen: (e: DragEvent) => void, onklasseraus: (klasse: string) => void, ontausch: (alt: string, neu: string) => void, onzusammen: () => void, ontrennen: () => void, oneinfuegen: () => void, onfest: () => void, onentfernen: () => void }} */
	let {
		zeile = $bindable(),
		i,
		platz,
		gezogen,
		ziel,
		markiert,
		tagGehtWeiter,
		marker,
		vorrat,
		onziehstart,
		onziehueber,
		onablegen,
		onklasseraus,
		ontausch,
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
	data-gruppe-weiter={tagGehtWeiter || undefined}
	ondragstart={onziehstart}
	ondragover={(e) => {
		e.preventDefault();
		onziehueber();
	}}
	ondrop={(e) => {
		e.preventDefault();
		onablegen(e);
	}}
	class="group h-12 transition-colors {flaeche} {ziel
		? '[&>td]:border-t-2 [&>td]:border-primary'
		: ''}"
>
	<!-- Der Griff erscheint unter der Maus (wie in Google Keep und Google Formulare) und zeigt,
	     dass man die Zeile ziehen kann; gezogen wird weiter an der ganzen Zeile. -->
	<td class="text-right tabular-nums text-on-surface-variant">
		<span class="flex items-center justify-between gap-2">
			<GripVertical
				class="h-4 w-4 cursor-grab opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100"
				aria-hidden="true"
			/>
			{i + 1}
		</span>
	</td>
	<LmfPlanPlatzZellen bind:zeile {i} {platz} {onfest} />
	<LmfPlanKlassenZelle klassen={zeile.klassen} {i} {vorrat} {marker} {onklasseraus} {ontausch} />
	<LmfPlanVermerkZelle bind:vermerk={zeile.vermerk} {i} ohneKlasse={zeile.klassen.length === 0} />
	<td class="px-4 py-1 text-right whitespace-nowrap">
		<LmfPlanZeileAktionen
			nummer={i + 1}
			klassen={zeile.klassen.length}
			fest={Boolean(zeile.fest)}
			platzlos={!platz}
			{onzusammen}
			{ontrennen}
			{oneinfuegen}
			{onfest}
			onklasseraus={() => onklasseraus(zeile.klassen[0])}
			{onentfernen}
		/>
	</td>
</tr>
