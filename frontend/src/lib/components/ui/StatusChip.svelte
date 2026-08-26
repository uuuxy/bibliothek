<script>
	// Ein Status-Chip nach Material 3.
	//
	// M3 kennt für „ein Zustand, den man nur liest" den Chip — nicht das Badge. Deshalb:
	// Radius 8 px (rounded-md, die Chip-Stufe der Shape-Skala; volle Pille wäre der
	// Button), Höhe 24 px, label-small in medium, Symbol 14 px, getönte Fläche statt
	// Rahmen. Der frühere Entwurf war ein winziges VERSALIENBADGE — laut, ohne Symbol,
	// und in der Lieferantenspalte vom truncate auf zwei Pixel zerquetscht.
	//
	// Farben kommen aus der Token-Schicht (theme-farben.css): emerald ist dort die
	// M3-Success-Familie, amber die Warning-Familie. Nichts wird hier frei gewählt.
	let { ton = 'neutral', text, detail = '', tip = '', icon = undefined } = $props();

	const toene = {
		// Getönte Container: Fläche -100, Schrift -700. Das ist das M3-Paar
		// container/on-container und hält den Kontrast auch bei kleinem Text.
		erfolg: 'bg-emerald-100 text-emerald-700',
		warten: 'bg-amber-100 text-amber-700',
		neutral: 'bg-slate-100 text-slate-600',
		// Etwas stimmt nicht (Meldung, Bestand reicht nicht): das M3-Paar
		// error-container / on-error-container aus der Rollenschicht.
		fehler: 'bg-error-container text-on-error-container'
	};
</script>

<span
	class="inline-flex h-6 shrink-0 items-center gap-1 rounded-md px-2 text-label-small font-semibold whitespace-nowrap {toene[
		ton
	]}"
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
