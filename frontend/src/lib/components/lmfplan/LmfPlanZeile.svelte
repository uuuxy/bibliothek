<!-- @component LmfPlanZeile — eine Zeile der Reihenfolge: Nummer, dann Wochentag,
     Datum und Stunde (gerechnet, nur lesbar — die zwei Spalten, die im Excel von Hand
     falsch waren; bei einer festgelegten Zeile Datumsfeld und Stunden-Auswahl,
     vorbelegt mit dem Vorschau-Platz), Klassen, Besonderheiten, Aktionen.

     Klassen nach M3: EINE Klasse steht als Text — „Don't display a single chip by
     itself" —, ab zwei Klassen (geteilte Stunde) ein Input-Chip-Set mit × je Chip.
     Ohne ersten Tag bleiben die gerechneten Spalten leer; der Abschnitt darüber sagt,
     was fehlt. Ein Status-Chip aus dem Marker (lmfplanDienst.klassenMarker): „ohne
     Schüler" an der Klasse. „Nur Rückgabe" ist seit 06.09.2026 Text im Vermerk (vom
     Vorschlag vorbelegt, änderbar), kein Chip.

     Ziel und Marke: `ziel` zeichnet beim Ziehen die Einfügelinie über der Zeile (Apple
     HIG: „display an insertion point … only when the destination can accept a dragged
     item"); `markiert` hebt eine gerade eingeplante Zeile kurz hervor (M3: Auswahl =
     secondary-container), damit man sieht, wohin die Nachbar-Regel sie gesetzt hat. -->
<script>
	import Feld from '../ui/Feld.svelte';
	import Select from '../ui/Select.svelte';
	import StatusChip from '../ui/StatusChip.svelte';
	import LmfKlasseChip from './LmfKlasseChip.svelte';
	import LmfPlanZeileAktionen from './LmfPlanZeileAktionen.svelte';
	import { STUNDEN, datumKurz, stundeText, wochentag } from '../../lmfplanDienst.js';

	/** @type {{ zeile: import('../../lmfplanDienst.js').PlanZeile, i: number, anzahl: number, platz: { datum: string, stunde: number } | undefined, gezogen: boolean, ziel: boolean, markiert: boolean, marker: ReturnType<typeof import('../../lmfplanDienst.js').klassenMarker>, onziehstart: () => void, onziehueber: () => void, onablegen: (e: DragEvent) => void, onklasseraus: (klasse: string) => void, onhoch: () => void, onrunter: () => void, onanfang: () => void, onende: () => void, onzusammen: () => void, ontrennen: () => void, oneinfuegen: () => void, onfest: () => void, onentfernen: () => void }} */
	let {
		zeile = $bindable(),
		i,
		anzahl,
		platz,
		gezogen,
		ziel,
		markiert,
		marker,
		onziehstart,
		onziehueber,
		onablegen,
		onklasseraus,
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

	const OHNE_SCHUELER_TIP =
		'Noch kein Schüler in dieser Klasse — sie kommt mit dem LUSD-Import oder gehört aus dem Plan';

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
	{#if zeile.fest}
		<td class="px-4 py-1 text-on-surface-variant" title="Fester Platz">
			{zeile.fest.datum ? wochentag(zeile.fest.datum) : ''}
		</td>
		<td class="px-4 py-1">
			<Feld
				id="lmf-zeile-fest-datum-{i}"
				aria-label="Fester Tag Zeile {i + 1}"
				type="date"
				bind:value={zeile.fest.datum}
				ungueltig={!zeile.fest.datum}
				feld="w-40"
			/>
		</td>
		<td class="px-4 py-1">
			<Select
				id="lmf-zeile-fest-stunde-{i}"
				aria-label="Feste Stunde Zeile {i + 1}"
				bind:value={zeile.fest.stunde}
				options={STUNDEN.map((st) => ({ value: st, label: `${st}. Std.` }))}
				class="w-28"
			/>
		</td>
	{:else}
		<td class="px-4 py-1 text-on-surface-variant">{platz ? wochentag(platz.datum) : ''}</td>
		<td class="px-4 py-1 tabular-nums text-on-surface">{platz ? datumKurz(platz.datum) : ''}</td>
		<td class="px-4 py-1 text-on-surface-variant">{platz ? stundeText(platz.stunde) : ''}</td>
	{/if}
	<td class="px-4 py-1">
		{#if zeile.klassen.length === 1}
			<span class="inline-flex items-center gap-2">
				<span class="font-medium text-on-surface">{zeile.klassen[0]}</span>
				{#if marker.ohneSchueler(zeile.klassen[0])}
					<StatusChip ton="warten" text="ohne Schüler" tip={OHNE_SCHUELER_TIP} />
				{/if}
			</span>
		{:else if zeile.klassen.length > 1}
			<div class="flex flex-wrap gap-1">
				{#each zeile.klassen as k (k)}
					<LmfKlasseChip
						name={k}
						hinweis={marker.ohneSchueler(k) ? 'ohne Schüler' : ''}
						onentfernen={() => onklasseraus(k)}
					/>
				{/each}
			</div>
		{/if}
	</td>
	<td class="px-4 py-1">
		<Feld
			id="lmf-zeile-vermerk-{i}"
			aria-label="Besonderheiten Zeile {i + 1}"
			bind:value={zeile.vermerk}
			placeholder={zeile.klassen.length === 0 ? 'Pflicht ohne Klasse' : ''}
			ungueltig={zeile.klassen.length === 0 && !zeile.vermerk.trim()}
		/>
	</td>
	<td class="px-4 py-1 text-right whitespace-nowrap">
		<LmfPlanZeileAktionen
			nummer={i + 1}
			{anzahl}
			klassen={zeile.klassen.length}
			fest={Boolean(zeile.fest)}
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
