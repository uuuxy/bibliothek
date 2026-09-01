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
	 */
	import { AlertTriangle, PackageCheck } from '@lucide/svelte';
	import { omniboxStore } from '../stores/omnibox.svelte.js';
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
