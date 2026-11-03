<script>
	// Ein Status-Chip nach Material 3.
	//
	// M3 kennt für „ein Zustand, den man nur liest" den Chip — nicht das Badge. Deshalb:
	// Radius 8 px (rounded-md, die Chip-Stufe der Shape-Skala; volle Pille wäre der
	// Button), Höhe 24 px, Symbol 14 px, getönte Fläche statt Rahmen.
	//
	// Farben sind die Paare container / on-container aus styles/rollen.css; sie halten den
	// Kontrast auch bei kleinem Text.
	//
	// punkt: ein Statuspunkt vor dem Text, wo die Fläche neutral bleibt (Bestand auf dem Cover
	// der Katalogkachel). vorlesen: der ganze Satz für Hilfstechnik, wenn der sichtbare Text eine
	// Kurzform ist. kuerzen: Der Text endet mit Auslassung, statt aus seinem Platz zu ragen; den
	// ganzen Text zeigt dann tip.
	let {
		ton = 'neutral',
		text,
		detail = '',
		tip = '',
		icon = undefined,
		punkt = undefined,
		vorlesen = '',
		kuerzen = false
	} = $props();

	const toene = {
		erfolg: 'bg-success-container text-on-success-container',
		warten: 'bg-warning-container text-on-warning-container',
		neutral: 'bg-surface-container text-on-surface-variant',
		// Etwas stimmt nicht (Meldung, Bestand reicht nicht).
		fehler: 'bg-error-container text-on-error-container'
	};

	const punkte = {
		erfolg: 'bg-success',
		warten: 'bg-warning',
		aus: 'bg-outline'
	};
</script>

<!-- data-chip: Marker für das Typo-Gate — Chips dürfen 12 px (dense), Zellen-Text nicht. -->
<span
	class="inline-flex h-6 items-center gap-1 rounded-md px-2 text-xs font-semibold whitespace-nowrap {kuerzen
		? 'min-w-0'
		: 'shrink-0'} {toene[ton]}"
	data-chip
	data-tip={tip}
>
	{#if punkt}
		<span class="size-2 shrink-0 rounded-full {punkte[punkt]}" aria-hidden="true"></span>
	{/if}
	{#if icon}
		{@const Symbol = icon}
		<Symbol size={14} strokeWidth={2.5} aria-hidden="true" class="shrink-0" />
	{/if}
	<!-- Ein zusammengesetzter String statt zweier Knoten: Zwischen {#if}-Blöcken frisst der
	     Formatierer die Leerzeichen, im Browser stand dann „Bestätigt  · 05.08." -->
	<span class:truncate={kuerzen} aria-hidden={vorlesen ? 'true' : undefined}
		>{detail ? `${text} · ${detail}` : text}</span
	>
	{#if vorlesen}
		<span class="sr-only">{vorlesen}</span>
	{/if}
</span>
