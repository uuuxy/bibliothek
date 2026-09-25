<!-- @component AuflagenZeile — eine Auflage in einer Liste (docs/OFFEN.md 4.18): links, woran
     man sie unterscheidet (Auflage und Jahr), darunter Titel, ISBN und Verlag, rechts der
     Bestand in denselben Worten wie im Katalog (bestandSatz). Material 3, Lists: Beschriftung
     mit Zusatztext, „Trailing text can provide additional meta-information about a list item,
     such as a price, count, or other details." Eine Zeile für den Abschnitt der Titelmaske und
     die Trefferliste des Zuordnen-Dialogs, damit beide dieselbe Auflage gleich nennen. -->
<script>
	import { bestandSatz } from '../../../../lib/utils/format.js';
	import { auflagenBeschriftung } from '../../../../lib/utils/auflagenText.js';

	/** @type {{ auflage?: string, erscheinungsjahr?: number, titel: string, isbn?: string, verlag?: string, gesamt?: number, verfuegbar?: number, imZulauf?: number, diese?: boolean }} */
	let {
		auflage,
		erscheinungsjahr,
		titel,
		isbn = '',
		verlag = '',
		gesamt,
		verfuegbar,
		imZulauf,
		diese = false
	} = $props();

	const zusatz = $derived([titel, isbn, verlag].filter(Boolean).join(' · '));

	// Das Leerzeichen vorn gehört zum Wert: Leerraum am Anfang eines Elements nimmt Svelte weg,
	// und Prettier bricht die Zeile genau dort um — am 25.09.2026 stand „2019— diese Auflage".
	// Eine Konstante statt {' — …'} im Markup, weil svelte/no-useless-mustaches den Literal
	// dort ablehnt.
	const DIESE_AUFLAGE = ' — diese Auflage';
</script>

<div class="flex min-w-0 flex-1 items-center gap-3">
	<div class="min-w-0 flex-1">
		<p class="truncate text-sm text-on-surface">
			{auflagenBeschriftung({ auflage, erscheinungsjahr })}{#if diese}<span
					class="text-on-surface-variant">{DIESE_AUFLAGE}</span
				>{/if}
		</p>
		<p class="truncate text-xs text-on-surface-variant">{zusatz}</p>
	</div>
	<span class="shrink-0 text-sm tabular-nums text-on-surface-variant"
		>{bestandSatz(gesamt, verfuegbar, imZulauf)}</span
	>
</div>
