<!-- @component BuchKarte — eine Kachel des Medienkatalogs als umrandete Karte nach Material 3
     (Cards, Specs: Fläche „Surface", Rahmen „Outline variant"). Beim Zeigen kommt nur die
     Zustandsschicht (m3-state) dazu; der Rahmen bleibt, wie er ist.

     Drei Zeilen mit fester Höhe, damit alle Kacheln einer Reihe gleich hoch sind und ISBN und
     Fußzeile auf einer Linie liegen: das Cover mit Titel, Bestand und Signatur
     (BuchKarteCover), der ISBN-Balken zum Kopieren, die Fußzeile mit Prüfdatum und Stift.
     Fach-, Klassen- und Zweig-Chips fehlen mit Absicht, das Suchfeld findet sie. Der Autor
     interessiert im Schulkatalog nicht. Ein Buch in mehreren Auflagen zählt die Summe; die
     Einblendung auf dem Cover sagt, woraus sie besteht (startseiten_api.js, buecherJeBuch). -->
<script module>
	/** Rahmen der Buchkacheln: umrandete Karte nach Material 3 (Cards, Specs: „Shape: 12dp
	 *  corner radius", Rahmen „Outline variant"). Katalog- und Klassenkachel nehmen ihn von
	 *  hier, damit sie nicht auseinanderlaufen. */
	export const KARTE = 'rounded-xl border border-outline-variant p-2';
</script>

<script>
	import BuchKarteCover from './BuchKarteCover.svelte';
	import Button from '../../../lib/components/ui/Button.svelte';
	import { formatDatum, jahrgangSpanne } from '../../../lib/utils/format.js';
	import { auflagenAufschluesselung } from '../../../lib/utils/auflagenText.js';
	import { kopiere } from '../../../lib/utils/zwischenablage.js';
	import { Check, Copy, SquarePen } from '@lucide/svelte';

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

	let copied = $state(false);

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

	/** @param {Event} e */
	async function copyIsbn(e) {
		e.stopPropagation();
		if (!titel.isbn || !(await kopiere(titel.isbn))) return;
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}
</script>

<!-- Die Karte ist für Hilfstechnik kein Knopf: Mit Kopieren-Knopf und Stift darin wäre sie
     ein Bedienelement im Bedienelement (axe: nested-interactive). Der Knopf ist das Cover.
     Der Klick auf die ganze Fläche bleibt als Komfort für Maus und Finger, ohne Rolle, deshalb
     die beiden Ausnahmen darunter. Die inneren Knöpfe stoppen die Weitergabe: Kopieren und
     Stift öffnen nicht auch die Akte (BuchKarte.test.js). -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<div
	class="m3-state group flex h-full cursor-pointer flex-col gap-2 {KARTE} text-on-surface"
	{onclick}
>
	<BuchKarteCover
		titel={titel.title}
		coverUrl={titel.coverUrl}
		isbn={titel.isbn}
		fach={fachZeile}
		auflage={titel.auflage}
		aufschluesselung={titel.buch ? auflagenAufschluesselung(titel.buch.auflagen) : ''}
		gesamt={zahlen.gesamt}
		verfuegbar={zahlen.verfuegbar}
		imZulauf={zahlen.imZulauf}
		signatur={titel.signatur}
	/>

	<!-- Die Nummer liest man gegen das Buch in der Hand ab, deshalb in Festbreite. Die Worte für
	     Hilfstechnik stehen unsichtbar daneben: Sie geben dem Balken seinen Namen, und eine Zahl
	     allein zählte für die Messung der Trefferflächen als Symbol. -->
	{#if titel.isbn}
		<Button type="button" variant="tonal" class="w-full" onclick={copyIsbn}>
			{#if copied}
				<span class="mr-auto flex items-center gap-1.5">
					<Check class="h-4 w-4" aria-hidden="true" />
					Kopiert!
				</span>
			{:else}
				<span class="sr-only">{nummerArt}</span>
				<span class="truncate font-mono font-normal">{titel.isbn}</span>
				<span class="sr-only">kopieren</span>
				<Copy class="ml-auto h-4 w-4 shrink-0" aria-hidden="true" />
			{/if}
		</Button>
	{:else}
		<!-- Ohne Fläche: Eine gefüllte runde Fläche hat in Material 3 die Form eines Knopfs, und hier
		     gibt es nichts zu kopieren. Die Höhe bleibt, damit Balken und Fußzeile in jeder Kachel auf
		     einer Linie liegen; eingerückt wie die Fußzeile. -->
		<div class="flex h-9 items-center pl-1 text-sm text-on-surface-variant">
			Keine {nummerArt}
		</div>
	{/if}

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
