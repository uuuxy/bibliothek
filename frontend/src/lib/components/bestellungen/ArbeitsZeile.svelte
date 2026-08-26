<script>
	/**
	 * @component ArbeitsZeile
	 * EINE Listenzeile für die Arbeitslisten des Bestell-Workspace (Klassensatz-
	 * Reservierungen, Wünsche & Meldungen) — nach der Material-3-„list item" mit
	 * führendem Element (Peters Entscheidung 26.08.2026).
	 *
	 * Die Frage, die die Bibliothek an diese Listen stellt, lautet in dieser Reihenfolge:
	 * WELCHE KLASSE will WELCHES BUCH, und WIE VIELE? Also:
	 *  - links der Kreis mit der Klasse (leading avatar: das, wonach man überfliegt),
	 *  - Headline = Titel (body-large 16), Supporting = Person · Datum (+ Chip),
	 *  - rechts die Anzahl groß (title-large 22) mit Beschriftung darunter,
	 *  - ganz rechts die Aktion als Snippet — die Liste entscheidet, was dort steht.
	 * Vorher stand die Klasse fett im Fließtext der zweiten Zeile, die Anzahl zwischen
	 * Punkten mittendrin, das Datum in einer eigenen Spalte.
	 *
	 * @prop {string} klasse
	 * @prop {string} titel
	 * @prop {string} neben - Supporting-Zeile (Person · Datum)
	 * @prop {{ text: string, ton?: string, tip?: string }} [chip] - Zustand rechts vom Nebentext
	 * @prop {{ text: string, ton?: string }} [art] - Chip an der Headline (Wunsch / Meldung)
	 * @prop {number} [anzahl]
	 * @prop {string} [anzahlLabel]
	 * @prop {string} [notiz] - Zitat der Lehrkraft, hinter dem Nebentext
	 * @prop {import('svelte').Snippet} aktion
	 */
	import StatusChip from '../ui/StatusChip.svelte';
	let {
		klasse,
		titel,
		neben,
		chip = undefined,
		art = undefined,
		anzahl = undefined,
		anzahlLabel = 'Exemplare',
		notiz = '',
		aktion
	} = $props();
</script>

<li class="flex items-center gap-4 py-3">
	<!-- Führendes Element: 48 px, wie der M3-Avatar. Die Klasse ist kurz (≤ 5 Zeichen),
	     Schrift label-large-Äquivalent (text-sm/500), damit auch „10g1" in den Kreis passt. -->
	<span
		class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-secondary-container text-sm font-semibold text-on-secondary-container"
		aria-label="Klasse {klasse}">{klasse}</span
	>
	<div class="min-w-0 flex-1">
		<p class="flex items-center gap-2 text-base font-bold text-on-surface">
			<span class="truncate">{titel}</span>
			{#if art}<StatusChip text={art.text} ton={art.ton ?? 'neutral'} />{/if}
		</p>
		<p class="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-on-surface-variant">
			<span class="truncate">{neben}{notiz ? ` — „${notiz}"` : ''}</span>
			{#if chip}<StatusChip
					text={chip.text}
					ton={chip.ton ?? 'neutral'}
					tip={chip.tip ?? ''}
				/>{/if}
		</p>
	</div>
	<!-- Feste Mindestbreite, damit die Zahlen aller Zeilen bündig stehen (tabular-nums). -->
	<div class="w-24 shrink-0 text-right">
		{#if anzahl != null}
			<p class="text-lg font-semibold tabular-nums leading-tight text-on-surface">{anzahl}</p>
			<p class="text-label-small uppercase tracking-wide text-on-surface-variant">{anzahlLabel}</p>
		{/if}
	</div>
	<div class="shrink-0">{@render aktion()}</div>
</li>
