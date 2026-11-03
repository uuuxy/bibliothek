<!-- @component BuchKarte — eine Kachel des Medienkatalogs, ohne Rahmen wie die Buchkacheln bei
     Google Play: Cover und Zeile darunter gehören durch den Abstand zusammen (KACHEL). Beim
     Zeigen hellt das Cover auf, und die Einblendung erscheint (BuchKarteCover).

     Zwei Zeilen: das Cover mit Titel, Bestand und Signatur (BuchKarteCover), darunter die
     Fußzeile mit Prüfdatum und Stift in fester Höhe, damit alle Kacheln einer Reihe gleich hoch
     sind. Die ISBN steht in der Einblendung auf dem Cover; kopiert wird sie in der Akte.
     Fach-, Klassen- und Zweig-Chips fehlen mit Absicht, das Suchfeld findet sie. Der Autor
     interessiert im Schulkatalog nicht. Ein Buch in mehreren Auflagen zählt die Summe; die
     Einblendung auf dem Cover sagt, woraus sie besteht (startseiten_api.js, buecherJeBuch). -->
<script module>
	/** Aufbau der Buchkacheln in Katalog, Klassensätzen und Portal: ohne Rahmen, das Cover und
	 *  die Zeile darunter dicht beieinander (Material 3, Spacing: „Implicit grouping uses close
	 *  proximity and open space (rather than lines and shadows) to group related items.").
	 *  8 px, weil die Trefferfläche des Stifts 6 px über die Zeile reicht und das Cover nicht
	 *  überdecken darf (e2e/icon-trefferflaechen.spec.js). */
	export const KACHEL = 'flex flex-col gap-2';

	/** Abstand der Kacheln im Raster: zwischen den Reihen mehr als zwischen Cover und Zeile, damit
	 *  die Zeile sichtbar zu ihrem Cover gehört. Alle drei Raster nehmen ihn von hier. */
	export const RASTER_ABSTAND = 'gap-x-4 gap-y-6';
</script>

<script>
	import BuchKarteCover from './BuchKarteCover.svelte';
	import { formatDatum, jahrgangSpanne } from '../../../lib/utils/format.js';
	import { auflagenAufschluesselung } from '../../../lib/utils/auflagenText.js';
	import { Check, SquarePen } from '@lucide/svelte';

	/**
	 * @type {{
	 *   titel: {
	 *     id: string,
	 *     isbn: string,
	 *     title: string,
	 *     author: string,
	 *     subject: string,
	 *     verfuegbar?: number,
	 *     gesamt?: number,
	 *     imZulauf?: number,
	 *     coverUrl: string,
	 *     lastCounted?: string,
	 *     signatur?: string,
	 *     medientyp?: string,
	 *     auflage?: string,
	 *     jahrgangVon?: number | string | null,
	 *     jahrgangBis?: number | string | null,
	 *     buch?: { gesamt: number, verfuegbar: number, imZulauf: number, auflagen: any[] }
	 *   },
	 *   onclick?: (event: Event) => void,
	 *   onEditClick?: () => void
	 * }}
	 */
	let { titel, onclick, onEditClick } = $props();

	const nummerArt = $derived(
		titel.medientyp === 'CD' || titel.medientyp === 'DVD' ? 'EAN' : 'ISBN'
	);

	const jahrgang = $derived(jahrgangSpanne(titel.jahrgangVon, titel.jahrgangBis));
	const fachZeile = $derived(
		[titel.subject, jahrgang && `Jahrgang ${jahrgang}`].filter(Boolean).join(' · ')
	);

	/** Ein Buch in mehreren Auflagen zählt die Summe. */
	const zahlen = $derived(titel.buch ?? titel);

	const geprueft = $derived(titel.lastCounted ? formatDatum(titel.lastCounted) : '');
</script>

<!-- Die Kachel ist für Hilfstechnik kein Knopf: Mit dem Stift darin wäre sie ein Bedienelement
     im Bedienelement (axe: nested-interactive). Der Knopf ist das Cover. Der Klick auf die ganze
     Fläche bleibt als Komfort für Maus und Finger, ohne Rolle, deshalb die beiden Ausnahmen
     darunter. Der Stift stoppt die Weitergabe: Er öffnet nicht auch die Akte (BuchKarte.test.js). -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<div class="group h-full cursor-pointer {KACHEL} text-on-surface" data-kachel {onclick}>
	<BuchKarteCover
		titel={titel.title}
		coverUrl={titel.coverUrl}
		isbn={titel.isbn}
		{nummerArt}
		fach={fachZeile}
		auflage={titel.auflage}
		aufschluesselung={titel.buch ? auflagenAufschluesselung(titel.buch.auflagen) : ''}
		gesamt={zahlen.gesamt}
		verfuegbar={zahlen.verfuegbar}
		imZulauf={zahlen.imZulauf}
		signatur={titel.signatur}
	/>

	<div class="flex h-9 items-center gap-2 pl-1 text-xs text-on-surface-variant">
		{#if geprueft}
			<span class="flex items-center gap-1" data-tip="Geprüft am {geprueft}">
				<Check class="h-3.5 w-3.5" aria-hidden="true" />
				<span class="sr-only">Geprüft</span>
				{geprueft}
			</span>
		{/if}
		{#if onEditClick}
			<button
				class="icon-btn ml-auto shrink-0"
				onclick={(e) => {
					e.stopPropagation();
					onEditClick();
				}}
				data-tip="Schnell bearbeiten"
				aria-label="Buch schnell bearbeiten"
			>
				<SquarePen class="h-4 w-4" aria-hidden="true" />
			</button>
		{/if}
	</div>
</div>
