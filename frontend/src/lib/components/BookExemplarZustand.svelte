<!-- @component BookExemplarZustand — was die Exemplar-Karte über den Zustand sagt:
     Menschentext (`zustand_notiz`), der erfasste Wertverlust in Prozent
     (`zustand_abwertung_prozent`, Migration 127) und der heutige Ersatzwert samt
     Herleitung (OFFEN.md 9.8, Stufe 2b).

     Eigenes Bauteil, weil die Karte an der 200-Zeilen-Ratsche steht: Die Zustandszeile
     wächst mit jeder neuen Angabe, die Karte darf es nicht.

     Die Zahl wird hier NICHT gerechnet. Sie kommt fertig vom Server, aus derselben
     Funktion wie der Vorschlag im Melde-Dialog — zwei Rechnungen an zwei Orten wären
     zwei Beträge für dasselbe Buch, und einer davon stünde in einem Bescheid. -->
<script>
	import { formatEuro } from '../utils/format.js';

	/** @type {{ ex: { zustand_notiz?: string, zustand_abwertung_prozent?: number, ersatzwert?: number, ersatzwert_herleitung?: string } }} */
	let { ex } = $props();

	const notiz = $derived(ex.zustand_notiz || '');
	const prozent = $derived(ex.zustand_abwertung_prozent ?? 0);
	const ersatzwert = $derived(ex.ersatzwert ?? 0);
	const herleitung = $derived(ex.ersatzwert_herleitung || '');
</script>

{#if notiz || prozent > 0}
	<p class="text-xs text-on-surface-variant">
		<span class="font-semibold">Zustand:</span>
		{notiz}
		{#if prozent > 0}
			<span class="font-semibold text-on-surface">{notiz ? '· ' : ''}{prozent} % Wertverlust</span>
		{/if}
	</p>
{/if}
{#if ersatzwert > 0}
	<!-- Der Betrag ohne die Herleitung wäre eine Behauptung: Erst der Satz macht ihn
	     nachrechenbar („3. Verleihjahr → 60 % von 41,50 €, abzüglich 20 %"). -->
	<p class="text-xs text-on-surface-variant">
		<span class="font-semibold">Ersatzwert heute:</span>
		<span class="font-semibold text-on-surface">{formatEuro(ersatzwert)}</span>
		{#if herleitung}
			<span class="block text-on-surface-variant">{herleitung}</span>
		{/if}
	</p>
{/if}
