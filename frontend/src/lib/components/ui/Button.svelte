<script>
	/** @type {{
	 *   children?: any,
	 *   variant?: 'primary' | 'secondary' | 'danger' | 'success' | 'danger-solid' | 'ghost',
	 *   size?: 'sm' | 'md' | 'lg',
	 *   class?: string,
	 *   element?: HTMLButtonElement,
	 *   [key: string]: any
	 * }} */
	let {
		children,
		variant = 'primary',
		size = 'md',
		class: className = '',
		element = $bindable(),
		...rest
	} = $props();

	// Ohne hover:bg-*: Die Rückmeldung beim Zeigen kommt aus dem State-Layer (.m3-state,
	// styles/komponenten.css) — eine Schicht in der Textfarbe über der unveränderten Fläche.
	//
	// Kein Schatten, in keiner Variante: In der M3-Token-Spezifikation (material-web v0.192)
	// trägt ein Bauteil entweder einen Rahmen oder eine Erhebung, nie beides.
	//
	// Farben nach M3, Buttons, Specs: gefüllt „Primary" mit „On primary"; umrandet „Outline
	// variant" mit „On surface variant". danger und success sind getönte Knöpfe in den
	// Container-Rollen von error und success, danger-solid der gefüllte in error.
	const variants = {
		primary: 'bg-primary text-on-primary border-transparent',
		secondary: 'bg-surface-container-lowest border-outline-variant text-on-surface-variant',
		danger: 'bg-error-container text-on-error-container border-transparent',
		'danger-solid': 'bg-error text-on-error border-transparent',
		success: 'bg-success-container text-on-success-container border-transparent',
		ghost: 'bg-transparent border-transparent text-on-surface-variant'
	};

	// Feste Höhen statt reinem Padding: Nur so stehen Buttons neben Eingabefeldern und in
	// Tabellenzeilen auf einer Linie. md = 36 px ist die gemeinsame Höhe der Bedienelemente.
	const sizes = {
		// M3 kennt keinen Knopf unter label-large (14 px).
		sm: 'h-8 px-2.5 text-sm',
		md: 'h-9 px-3 text-sm',
		lg: 'h-10 px-4 text-sm'
	};

	// rounded-full: In Material 3 ist der Button eine Pille (Shape-Skala: Menüs 4 px, Chips 8,
	// Karten 12, Dialoge 28, Buttons voll).
	//
	// Gesperrt: ein Zustand für alle Varianten, nach Material 3 — Fläche in der Textfarbe bei
	// 12 %, Beschriftung bei 38 %, kein Rahmen, kein Schatten. Mit `!`, weil die Klassen der
	// Variante dieselbe Spezifität haben und dann die Reihenfolge im Stylesheet entschiede.
	const disabledClasses =
		'disabled:cursor-not-allowed disabled:bg-on-surface/12! disabled:text-on-surface/38! disabled:border-transparent! disabled:shadow-none!';

	const baseClasses = `m3-state inline-flex items-center justify-center gap-2 font-semibold transition-colors border rounded-full cursor-pointer ${disabledClasses} focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2`;

	// Farb-Utilities des Aufrufers ersetzen die der Variante, statt mit ihnen zu konkurrieren:
	// Tailwind-Utilities haben alle dieselbe Spezifität, also entschiede sonst die Reihenfolge
	// im Stylesheet und nicht die im class-Attribut. Nur Basisfarben werden ersetzt;
	// Zustandsvarianten (hover:, disabled:, focus-within:) bleiben stehen, und Größenangaben
	// (text-label-small, text-sm) gelten nicht als Farbe.
	const FARBE =
		/^(bg|border|ring|text)-(slate|gray|zinc|blue|indigo|emerald|green|amber|orange|rose|red|white|black|transparent|primary|secondary|tertiary|error|success|warning|surface|outline|scrim|inverse-|on-)/;
	const familie = (/** @type {string} */ c) => c.split('-')[0];

	const variantClasses = $derived.by(() => {
		const eigene = new Set(
			className
				.split(/\s+/)
				.filter((c) => FARBE.test(c))
				.map(familie)
		);
		if (eigene.size === 0) return variants[variant];
		return variants[variant]
			.split(/\s+/)
			.filter((c) => !(FARBE.test(c) && eigene.has(familie(c))))
			.join(' ');
	});
</script>

<button
	bind:this={element}
	class="{baseClasses} {sizes[size]} {variantClasses} {className}"
	{...rest}
>
	{@render children?.()}
</button>
