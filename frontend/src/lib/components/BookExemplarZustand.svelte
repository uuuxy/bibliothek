<!-- @component BookExemplarZustand — die Zustandszeile einer Exemplar-Karte:
     Menschentext (`zustand_notiz`) und der erfasste Wertverlust in Prozent
     (`zustand_abwertung_prozent`, Migration 127).

     Eigenes Bauteil, weil die Karte an der 200-Zeilen-Ratsche steht: Die Zeile
     wächst mit jeder neuen Zustandsangabe, die Karte darf es nicht. -->
<script>
	/** @type {{ ex: { zustand_notiz?: string, zustand_abwertung_prozent?: number } }} */
	let { ex } = $props();

	const notiz = $derived(ex.zustand_notiz || '');
	const prozent = $derived(ex.zustand_abwertung_prozent ?? 0);
</script>

{#if notiz || prozent > 0}
	<p class="text-xs text-on-surface-variant">
		<span class="font-semibold">Zustand:</span>
		{notiz}
		<!-- Der Wertverlust steht hier, weil er den Ersatzbetrag mindert: Wer ihn nicht
		     sieht, hält den vorgeschlagenen Betrag für zu niedrig. -->
		{#if prozent > 0}
			<span class="font-semibold text-on-surface">{notiz ? '· ' : ''}{prozent} % Wertverlust</span>
		{/if}
	</p>
{/if}
