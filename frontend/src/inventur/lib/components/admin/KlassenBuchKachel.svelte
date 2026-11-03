<!-- @component KlassenBuchKachel — ein Titel im Klassensatz, gebaut wie die Katalog-Kachel
     (BuchKarte): umrandete Karte, der Bestand als Chip auf dem Cover, darunter der Titel und die
     Hinweise der Klasse. Die stehen immer da, nicht erst beim Zeigen wie im Katalog, denn am
     Tablet gibt es kein Zeigen. ISBN-Balken, Prüfdatum und Stift fehlen: Hier zählt, was die
     Klasse hat und ob der Bestand reicht. -->
<script>
	import BuchCover from '../../../../lib/components/ui/BuchCover.svelte';
	import BestandChip from '../BestandChip.svelte';
	import { KARTE } from '../BuchKarte.svelte';
	import {
		auflagenDerKlasse,
		auflagenAufschluesselung
	} from '../../../../lib/utils/auflagenText.js';

	/**
	 * @type {{
	 *   titel: {
	 *     id: string,
	 *     isbn: string,
	 *     title: string,
	 *     subject: string,
	 *     coverUrl: string,
	 *     verfuegbar?: number,
	 *     gesamt?: number,
	 *     imZulauf?: number,
	 *     quelle?: string,
	 *     leser?: number,
	 *     auflagen?: { id: string, auflage?: string, erscheinungsjahr?: number, kinder: number }[],
	 *     auflagenBestand?: { auflage?: string, erscheinungsjahr?: number, gesamt_bestand: number }[]
	 *   },
	 *   onEdit?: (book: any) => void,
	 *   bearbeitbar?: boolean
	 * }}
	 * bearbeitbar=false: reine Ansicht (Kollegiums-Portal). Ohne den Schalter
	 * wäre jede Kachel dort ein Knopf, hinter dem für die Rolle ein 403 liegt.
	 */
	let { titel, onEdit = undefined, bearbeitbar = true } = $props();
</script>

<!-- Zwei Hüllen, ein Inhalt: Die Rolle „button" muss für svelte-check STATISCH am Element
     stehen (a11y_no_noninteractive_tabindex). Der Inhalt liegt einmal im Snippet. -->
{#snippet inhalt()}
	<!-- Der Chip liegt unten auf dem Cover wie in der Katalog-Kachel (BuchKarteCover). -->
	<div class="relative">
		<BuchCover
			coverUrl={titel.coverUrl}
			isbn={titel.isbn}
			titel={titel.title}
			fach={titel.subject}
			groesse="kachel"
			dekorativ
		/>
		<div class="absolute inset-x-2 bottom-2 flex">
			<BestandChip gesamt={titel.gesamt} verfuegbar={titel.verfuegbar} imZulauf={titel.imZulauf} />
		</div>
	</div>
	<div class="flex flex-col gap-0.5 text-sm">
		<!-- text-base ausdrücklich am Titel: Sonst erbt er die 14 px vom Container darüber,
		     und eine geerbte Größe sieht kein Grep, nur die Messung im Browser. -->
		<h3
			class="line-clamp-2 text-base leading-snug font-semibold wrap-break-word text-on-surface"
			title={titel.title}
		>
			{titel.title}
		</h3>
		{#if titel.auflagenBestand?.length}
			<!-- Ein Buch in mehreren Auflagen (Portal „Schulbücher", docs/OFFEN.md 4.18, Stufe 6):
			     Die Zahlen darüber sind die Summe; hier steht, woraus sie besteht — wie auf der
			     Kachel des Medienkatalogs. -->
			<span class="text-xs text-on-surface-variant" data-testid="schulbuch-auflagen"
				>{auflagenAufschluesselung(titel.auflagenBestand)}</span
			>
		{/if}
		{#if titel.quelle === 'ausleihe'}
			<!-- Live aus den Ausleihen abgeleitet, nicht gespeichert: Mehr als die Hälfte der
			     Klasse hat den Titel. Die Handliste bleibt unangetastet. -->
			<span class="text-xs text-on-surface-variant" data-testid="klassensatz-aus-ausleihen">
				aus Ausleihen · {titel.leser} Leser
			</span>
		{/if}
		{#if titel.auflagen?.length}
			<!-- Gemischte Auflagen (docs/OFFEN.md 4.18, Stufe 5): Kinder der Klasse haben eine
			     andere Auflage als die der Kachel — andere Auflagen haben andere Seitenzahlen.
			     Begleittext der Karte (M3 Cards, Anatomie: „Supporting text"), in der Warnrolle
			     wie die Zeile an der Theke; auf der Fläche die Rolle selbst, nicht der Container. -->
			<div class="text-xs text-warning" data-testid="klassensatz-auflagen">
				<p>Auflagen in der Klasse</p>
				<ul>
					{#each auflagenDerKlasse(titel.auflagen, titel.id) as zeile, i (i)}
						<li>{zeile}</li>
					{/each}
				</ul>
			</div>
		{/if}
	</div>
{/snippet}

{#if bearbeitbar}
	<div
		class="m3-state flex cursor-pointer flex-col gap-2 {KARTE}"
		onclick={() => onEdit?.(titel)}
		role="button"
		tabindex="0"
		onkeydown={(e) => {
			if (e.target !== e.currentTarget) return;
			if (e.key === 'Enter' || e.key === ' ') {
				e.preventDefault();
				onEdit?.(titel);
			}
		}}
	>
		{@render inhalt()}
		<!-- Der Name des Knopfs kommt aus seinem Inhalt und endet mit dem, was er tut: Ein fremder
		     aria-label verschwiege, was auf der Kachel steht (WCAG 2.5.3, Beschriftung im Namen). -->
		<span class="sr-only">bearbeiten</span>
	</div>
{:else}
	<div class="flex flex-col gap-2 {KARTE}">
		{@render inhalt()}
	</div>
{/if}
