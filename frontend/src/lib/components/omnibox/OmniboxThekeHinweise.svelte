<script>
	/**
	 * @component OmniboxThekeHinweise
	 * Die Hinweis-Banner über dem aktiven Schülerkonto an der Theke, gerendert nur mit
	 * aktivem Schüler.
	 *
	 * 1. Fremdrückgabe: Das gescannte Buch war auf jemand anderen verbucht.
	 * 2. Abholfach: Schüler scannen nicht selbst, der Hinweis sagt der Mitarbeiterin, dass
	 *    für den gerade gescannten Schüler ein vorgemerktes Buch im Abholfach liegt. Ohne
	 *    ihn stünde der Schüler an der Theke, während sein Buch im Fach auf den Ablauf der
	 *    Abholfrist wartet (repository.BestimmeAbholfrist).
	 * 3. Kollegium: Die Theke zeigt einem Kollegen dieselbe Akte wie einem Schüler, nur ohne
	 *    Klasse. Der Hinweis sagt, was daran anders ist: Die Frist ist ein Jahr
	 *    (ladeVorgang).
	 * 4. Gemischte Auflagen: Das eben ausgeliehene Schulbuch ist eine andere Auflage als
	 *    die, die Kinder derselben Klasse schon haben, und verschiedene Auflagen heißen
	 *    verschiedene Seitenzahlen. Eine Zeile wie bei der Fremdrückgabe, kein Dialog: Die
	 *    Ausleihe ist gebucht, wer will, legt das Buch zurück und holt die andere Auflage.
	 */
	import { TriangleAlert, BookCopy, PackageCheck, GraduationCap } from '@lucide/svelte';
	import { omniboxStore } from '../../stores/omnibox.svelte.js';
	import { leserArtText, istKollegium } from '../leser/leserArt.js';
	import { auflagenHinweisText } from '../../utils/auflagenText.js';
</script>

{#if omniboxStore.letzteFremdrueckgabe}
	<!-- Eine Betonung, nicht vier: Der entscheidende Teil ist, auf wen NICHT
	     gebucht wurde. Wenn jedes zweite Wort fett ist, betont keines mehr. -->
	<div
		class="bg-warning-container text-on-warning-container no-print mb-2 flex w-full max-w-xl items-center space-x-2 p-3 text-xs"
	>
		<TriangleAlert class="h-4 w-4 shrink-0" aria-hidden="true" />
		<span
			>Fremdrückgabe: Buch war auf {omniboxStore.letzteFremdrueckgabe.vorbesitzerName} verbucht und wurde
			dort zurückgegeben —
			<strong class="font-medium">nicht auf {omniboxStore.aktiverLeser.vorname} gebucht</strong>.
			Erneut scannen, um es auszuleihen.</span
		>
	</div>
{/if}

{#if omniboxStore.letzterAuflagenHinweis}
	<div
		class="bg-warning-container text-on-warning-container no-print mb-2 flex w-full max-w-xl items-center space-x-2 p-3 text-xs"
		role="status"
	>
		<BookCopy class="h-4 w-4 shrink-0" aria-hidden="true" />
		<span>{auflagenHinweisText(omniboxStore.letzterAuflagenHinweis)}</span>
	</div>
{/if}

{#if omniboxStore.abholbereit.length > 0}
	<div
		class="bg-primary-container text-on-primary-container no-print mb-2 flex w-full max-w-xl items-center space-x-2 p-3 text-xs"
	>
		<PackageCheck class="h-4 w-4 shrink-0" aria-hidden="true" />
		<span>
			Abholfach: Für {omniboxStore.aktiverLeser.vorname} liegt bereit —
			{#each omniboxStore.abholbereit as v, i (i)}
				{i > 0 ? ' · ' : ''}<strong class="font-medium">„{v.titel}"</strong>{v.bereitgestellt_bis
					? ` (bis ${new Date(v.bereitgestellt_bis).toLocaleDateString('de-DE')})`
					: ''}
			{/each}
			— bitte direkt mitgeben.
		</span>
	</div>
{/if}

{#if istKollegium(omniboxStore.aktiverLeser)}
	<div
		class="bg-secondary-container text-on-secondary-container no-print mb-2 flex w-full max-w-xl items-center space-x-2 p-3 text-xs"
	>
		<GraduationCap class="h-4 w-4 shrink-0" aria-hidden="true" />
		<span>
			{leserArtText(omniboxStore.aktiverLeser.art)} geladen — gescannte Bücher gehen auf
			<strong class="font-medium">{omniboxStore.aktiverLeser.vorname}</strong>, Frist ein Jahr.
		</span>
	</div>
{/if}
