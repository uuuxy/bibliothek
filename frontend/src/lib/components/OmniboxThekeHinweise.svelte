<script>
	/**
	 * @component OmniboxThekeHinweise
	 * Die Hinweis-Banner über dem aktiven Schülerkonto an der Theke — aus
	 * Omnibox.svelte extrahiert (200-Zeilen-Regel), gerendert nur mit aktivem Schüler.
	 *
	 * 1. Fremdrückgabe: Das gescannte Buch war auf jemand anderen verbucht.
	 * 2. Abholfach (Betreiber-Entscheidung 01.09.2026): Schüler scannen nicht
	 *    selbst — der Hinweis sagt der MITARBEITERIN, dass für den gerade
	 *    gescannten Schüler ein vorgemerktes Buch im Abholfach liegt. Ohne ihn
	 *    stünde der Schüler an der Theke, während sein Buch im Fach auf den
	 *    Ablauf der 3-Tage-Frist wartet.
	 * 3. Kollegium (16.09.2026): Seit die Akte auch einen Kollegen zeigt, sieht die
	 *    Theke dieselbe Ansicht wie bei einem Schüler — nur ohne Klasse. Der
	 *    Hinweis sagt, was daran anders ist: Die Frist ist ein Jahr
	 *    (resolveBorrowerAndDueTime). Vorher stand dafür eine eigene schmale
	 *    Karte da, weil es die Akte noch nicht gab.
	 * 4. Gemischte Auflagen (25.09.2026, docs/OFFEN.md 4.18, Stufe 5): Das eben
	 *    ausgeliehene Schulbuch ist eine andere Auflage als die, die Kinder derselben
	 *    Klasse schon haben — verschiedene Auflagen heißen verschiedene Seitenzahlen.
	 *    Entschieden: eine Zeile wie bei der Fremdrückgabe, kein Dialog; die Ausleihe
	 *    ist gebucht, wer will, legt das Buch zurück und holt die andere Auflage.
	 */
	import { AlertTriangle, BookCopy, PackageCheck, GraduationCap } from '@lucide/svelte';
	import { omniboxStore } from '../stores/omnibox.svelte.js';
	import { leserArtText, istKollegium } from '../leserArt.js';
	import { auflagenHinweisText } from '../utils/auflagenText.js';
</script>

{#if omniboxStore.lastFremdrueckgabe}
	<!-- Eine Betonung, nicht vier: Der entscheidende Teil ist, auf wen NICHT
	     gebucht wurde. Wenn jedes zweite Wort fett ist, betont keines mehr. -->
	<div
		class="no-print mb-2 flex w-full max-w-xl items-center space-x-2 border border-amber-100 bg-amber-50 p-3 text-xs text-amber-800"
	>
		<AlertTriangle class="h-4 w-4 shrink-0" aria-hidden="true" />
		<span
			>Fremdrückgabe: Buch war auf {omniboxStore.lastFremdrueckgabe.vorbesitzerName} verbucht und wurde
			dort zurückgegeben —
			<strong class="font-medium">nicht auf {omniboxStore.activeStudent.vorname} gebucht</strong>.
			Erneut scannen, um es auszuleihen.</span
		>
	</div>
{/if}

{#if omniboxStore.lastAuflagenHinweis}
	<div
		class="bg-warning-container text-on-warning-container no-print mb-2 flex w-full max-w-xl items-center space-x-2 p-3 text-xs"
		role="status"
	>
		<BookCopy class="h-4 w-4 shrink-0" aria-hidden="true" />
		<span>{auflagenHinweisText(omniboxStore.lastAuflagenHinweis)}</span>
	</div>
{/if}

{#if omniboxStore.abholbereit.length > 0}
	<div
		class="bg-primary-container text-on-primary-container no-print mb-2 flex w-full max-w-xl items-center space-x-2 p-3 text-xs"
	>
		<PackageCheck class="h-4 w-4 shrink-0" aria-hidden="true" />
		<span>
			Abholfach: Für {omniboxStore.activeStudent.vorname} liegt bereit —
			{#each omniboxStore.abholbereit as v, i (i)}
				{i > 0 ? ' · ' : ''}<strong class="font-medium">„{v.titel}"</strong>{v.bereitgestellt_bis
					? ` (bis ${new Date(v.bereitgestellt_bis).toLocaleDateString('de-DE')})`
					: ''}
			{/each}
			— bitte direkt mitgeben.
		</span>
	</div>
{/if}

{#if istKollegium(omniboxStore.activeStudent)}
	<div
		class="bg-secondary-container text-on-secondary-container no-print mb-2 flex w-full max-w-xl items-center space-x-2 p-3 text-xs"
	>
		<GraduationCap class="h-4 w-4 shrink-0" aria-hidden="true" />
		<span>
			{leserArtText(omniboxStore.activeStudent.art)} geladen — gescannte Bücher gehen auf
			<strong class="font-medium">{omniboxStore.activeStudent.vorname}</strong>, Frist ein Jahr.
		</span>
	</div>
{/if}
