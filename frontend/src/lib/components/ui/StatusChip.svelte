<script>
	// Ein Status-Chip nach Material 3.
	//
	// M3 kennt für „ein Zustand, den man nur liest" den Chip — nicht das Badge. Deshalb:
	// Radius 8 px (rounded-md, die Chip-Stufe der Shape-Skala; volle Pille wäre der
	// Button), Höhe 24 px, Symbol 14 px, getönte Fläche statt Rahmen.
	//
	// Farben sind die Paare container / on-container aus styles/rollen.css; sie halten den
	// Kontrast auch bei kleinem Text.
	let { ton = 'neutral', text, detail = '', tip = '', icon = undefined } = $props();

	const toene = {
		erfolg: 'bg-success-container text-on-success-container',
		warten: 'bg-warning-container text-on-warning-container',
		neutral: 'bg-surface-container text-on-surface-variant',
		// Etwas stimmt nicht (Meldung, Bestand reicht nicht).
		fehler: 'bg-error-container text-on-error-container'
	};
</script>

<!-- data-chip: Marker für das Typo-Gate — Chips dürfen 12 px (dense), Zellen-Text nicht. -->
<span
	class="inline-flex h-6 shrink-0 items-center gap-1 rounded-md px-2 text-xs font-semibold whitespace-nowrap {toene[
		ton
	]}"
	data-chip
	data-tip={tip}
>
	{#if icon}
		{@const Symbol = icon}
		<Symbol size={14} strokeWidth={2.5} aria-hidden="true" />
	{/if}
	<!-- Ein zusammengesetzter String statt zweier Knoten: Zwischen {#if}-Blöcken frisst der
	     Formatierer die Leerzeichen, im Browser stand dann „Bestätigt  · 05.08." -->
	{detail ? `${text} · ${detail}` : text}
</span>
