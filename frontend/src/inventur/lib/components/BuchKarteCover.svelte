<!-- @component BuchKarteCover — das Cover der Katalogkachel mit Titel, Bestand und Standort.

     Text auf einem Bild braucht nach Material 3 eine Unterlage (Cards, Guidelines: „Add a
     translucent scrim or bounding shape beneath the text or icon to help ensure proper
     contrast"). Der Titel liegt deshalb beim Zeigen auf einer Fläche im Surface-Ton, die im
     hellen wie im dunklen Design zum Text passt, und die Chips sind deckend. Ohne Bild trägt
     der Platzhalter von BuchCover Fach, Jahrgang und Titel. -->
<script>
	import BuchCover, { PLATZHALTER } from '../../../lib/components/ui/BuchCover.svelte';
	import StatusChip from '../../../lib/components/ui/StatusChip.svelte';
	import { MapPin } from '@lucide/svelte';

	/**
	 * @type {{
	 *   titel: string,
	 *   coverUrl?: string,
	 *   isbn?: string,
	 *   fach?: string,
	 *   auflage?: string,
	 *   aufschluesselung?: string,
	 *   bestand: string,
	 *   bestandSatz: string,
	 *   lage: 'alle' | 'teil' | 'keins',
	 *   signatur?: string
	 * }}
	 */
	let {
		titel,
		coverUrl = '',
		isbn = '',
		fach = '',
		auflage = '',
		aufschluesselung = '',
		bestand,
		bestandSatz,
		lage,
		signatur = ''
	} = $props();

	/** Grün, wenn alles im Regal steht, gelb bei einem Teil. Nichts im Regal ist grau, nicht
	 *  rot: Im Schuljahr ist fast jedes Lernmittel verliehen, und ein überall roter Katalog
	 *  sagt nichts mehr. */
	const PUNKT = { alle: 'erfolg', teil: 'warten', keins: 'aus' };
</script>

<div class="group/cover relative">
	<!-- Der Knopf trägt die Semantik (Tastatur, Screenreader) mit dem Titel als Namen, die
	     Überschrift macht die Kacheln für Hilfstechnik einzeln anspringbar. Einen eigenen
	     Handler hat er nicht: Sein Klick steigt zur Karte auf und öffnet dort die Akte, ein
	     Handler hier navigierte doppelt. Die Zustandsschicht gibt die Karte, nicht der Knopf. -->
	<h2>
		<button
			type="button"
			aria-label={titel}
			class="no-state-layer block w-full cursor-pointer rounded-lg focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:outline-none"
		>
			<BuchCover {coverUrl} {isbn} {titel} {fach} groesse="kachel" dekorativ />
		</button>
	</h2>

	<!-- Die Einblendung setzt Fach und Titel in den Klassen des Platzhalters von BuchCover
	     (PLATZHALTER) und deckt: Ohne Bild verlängert sie den Platzhalter nur um Auflage und
	     Aufschlüsselung, statt den Titel ein zweites Mal versetzt darüberzulegen. Ohne
	     Zeigerereignisse, damit ein Klick auf sie den Knopf darunter trifft. -->
	<div
		class="pointer-events-none absolute inset-x-0 top-0 max-h-full overflow-hidden rounded-t-lg {PLATZHALTER.flaeche} text-left opacity-0 transition-opacity group-hover:opacity-100 group-has-[:focus-visible]/cover:opacity-100"
	>
		<div aria-hidden="true">
			<p class={PLATZHALTER.fach}>{fach}</p>
			<p class={PLATZHALTER.titel}>{titel}</p>
		</div>
		<!-- Die Auflage gehört zum Titel: Zwei Kacheln „Lambacher Schweizer 7" sind zwei
		     verschiedene Bücher, und das sieht man sonst nirgends. -->
		{#if auflage}
			<p class="mt-1.5 text-xs text-on-surface-variant">{auflage}</p>
		{/if}
		{#if aufschluesselung}
			<p class="mt-1.5 text-xs text-on-surface-variant">{aufschluesselung}</p>
		{/if}
	</div>

	<div class="absolute inset-x-2 bottom-2 flex items-center gap-2">
		{#if bestand}
			<StatusChip
				text={bestand}
				punkt={PUNKT[lage]}
				vorlesen={bestand === bestandSatz ? '' : bestandSatz}
				tip={bestandSatz}
			/>
		{/if}
		{#if signatur}
			<span class="ml-auto flex min-w-0">
				<StatusChip text={signatur} icon={MapPin} tip="Signatur {signatur}" kuerzen />
			</span>
		{/if}
	</div>
</div>
