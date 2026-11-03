<!-- @component BuchKarteCover — das Cover der Katalogkachel mit Titel, Bestand und Standort.

     Text auf einem Bild braucht nach Material 3 eine Unterlage (Cards, Guidelines: „Add a
     translucent scrim or bounding shape beneath the text or icon to help ensure proper
     contrast"). Der Titel liegt deshalb beim Zeigen auf einer Fläche im Surface-Ton, die im
     hellen wie im dunklen Design zum Text passt, und die Chips sind deckend. Ohne Bild trägt
     der Platzhalter von BuchCover Fach, Jahrgang und Titel. -->
<script>
	import BuchCover from '../../../lib/components/ui/BuchCover.svelte';
	import StatusChip from '../../../lib/components/ui/StatusChip.svelte';
	import BestandChip from './BestandChip.svelte';
	import CoverEinblendung from './CoverEinblendung.svelte';
	import { MapPin } from '@lucide/svelte';

	/**
	 * @type {{
	 *   titel: string,
	 *   coverUrl?: string,
	 *   isbn?: string,
	 *   nummerArt?: string,
	 *   fach?: string,
	 *   auflage?: string,
	 *   aufschluesselung?: string,
	 *   gesamt?: number,
	 *   verfuegbar?: number,
	 *   imZulauf?: number,
	 *   signatur?: string
	 * }}
	 */
	let {
		titel,
		coverUrl = '',
		isbn = '',
		nummerArt = 'ISBN',
		fach = '',
		auflage = '',
		aufschluesselung = '',
		gesamt,
		verfuegbar,
		imZulauf,
		signatur = ''
	} = $props();
</script>

<div class="group/cover relative">
	<!-- Der Knopf trägt die Semantik (Tastatur, Screenreader) mit dem Titel als Namen, die
	     Überschrift macht die Kacheln für Hilfstechnik einzeln anspringbar. Einen eigenen
	     Handler hat er nicht: Sein Klick steigt zur Kachel auf und öffnet dort die Akte, ein
	     Handler hier navigierte doppelt. Die Zustandsschicht beim Zeigen trägt er selbst: Die
	     Kachel hat keinen Rahmen, den sie füllen könnte. -->
	<h2>
		<button
			type="button"
			aria-label={titel}
			class="block w-full cursor-pointer rounded-lg focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:outline-none"
		>
			<BuchCover {coverUrl} {isbn} {titel} {fach} groesse="kachel" dekorativ />
		</button>
	</h2>

	<!-- Die Auflage gehört zum Titel: Zwei Kacheln „Lambacher Schweizer 7" sind zwei verschiedene
	     Bücher, und das sieht man sonst nirgends. -->
	<CoverEinblendung {titel} {fach} zeilen={[auflage, aufschluesselung]} nummer={isbn} {nummerArt} />

	<div class="absolute inset-x-2 bottom-2 flex items-center gap-2">
		<BestandChip {gesamt} {verfuegbar} {imZulauf} />
		{#if signatur}
			<span class="ml-auto flex min-w-0">
				<StatusChip text={signatur} icon={MapPin} tip="Signatur {signatur}" kuerzen />
			</span>
		{/if}
	</div>
</div>
