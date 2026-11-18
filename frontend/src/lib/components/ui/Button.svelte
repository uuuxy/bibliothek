<script>
	/** @type {{
	 *   children?: any,
	 *   variant?: 'primary' | 'secondary' | 'tonal' | 'danger' | 'success' | 'danger-solid' | 'ghost' | 'symbol',
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
	// variant" mit „On surface variant"; tonal „Secondary container" mit „On secondary
	// container"; der Textknopf (ghost) „Primary". danger und success sind getönte Knöpfe in
	// den Container-Rollen von error und success, danger-solid der gefüllte in error.
	//
	// ghost steht in der Hauptfarbe, weil er keine Fläche hat (M3 Buttons, Guidelines): „the
	// label text color must always be recognizable from non-button text and elements". symbol
	// ist der Knopf, der nur ein Symbol trägt (M3 Icon buttons, Specs: „Standard icon: On
	// surface variant"); Button.test.js hält die beiden auseinander. Mit .icon-btn bekommt er
	// die Trefferfläche von 48 px wie jeder Symbolknopf.
	const variants = {
		primary: 'bg-primary text-on-primary border-transparent',
		secondary: 'bg-surface-container-lowest border-outline-variant text-on-surface-variant',
		tonal: 'bg-secondary-container text-on-secondary-container border-transparent',
		danger: 'bg-error-container text-on-error-container border-transparent',
		'danger-solid': 'bg-error text-on-error border-transparent',
		success: 'bg-success-container text-on-success-container border-transparent',
		ghost: 'bg-transparent border-transparent text-primary',
		symbol: 'icon-btn bg-transparent border-transparent text-on-surface-variant'
	};

	// Feste Höhen statt reinem Padding: Nur so stehen Buttons neben Eingabefeldern und in
	// Tabellenzeilen auf einer Linie. md = 36 px ist die gemeinsame Höhe der Bedienelemente.
	const sizes = {
		// M3 kennt keinen Knopf unter label-large (14 px).
		sm: 'h-8 px-2.5 text-label-large',
		md: 'h-9 px-3 text-label-large',
		lg: 'h-10 px-4 text-label-large'
	};
	// Ein Symbolknopf ist quadratisch ohne den Innenabstand eines Knopfs mit Text, M3 Icon
	// buttons: Container „extra small“ 32 dp mit Icon 20 dp, Breite „Default“.
	const symbolSizes = { sm: 'size-8', md: 'size-9', lg: 'size-10' };

	// rounded-full: In Material 3 ist der Button eine Pille (Shape-Skala: Menüs 4 px, Chips 8,
	// Karten 12, Dialoge 28, Buttons voll).
	//
	// Gesperrt nach Material 3: Beschriftung und Symbol in on-surface bei 38 %, kein Schatten. Eine
	// Fläche in on-surface bei 12 % behält nur ein Knopf mit Fläche; Text- und Symbolknopf bleiben
	// ohne, der umrandete behält den Rand bei 12 % (Googles Token in material-web: text-button
	// und icon-button kennen nur disabled-label-text und disabled-icon, outlined-button dazu
	// disabled-outline). Mit `!`, weil die Klassen der Variante dieselbe Spezifität haben und dann
	// die Reihenfolge im Stylesheet entschiede.
	const disabledClasses =
		'disabled:cursor-not-allowed disabled:text-on-surface/38! disabled:shadow-none!';
	const flaecheGesperrt = 'disabled:bg-on-surface/12! disabled:border-transparent!';
	/** @type {Record<string, string>} */
	const ohneFlaecheGesperrt = {
		ghost: 'disabled:bg-transparent! disabled:border-transparent!',
		symbol: 'disabled:bg-transparent! disabled:border-transparent!',
		secondary: 'disabled:bg-transparent! disabled:border-on-surface/12!'
	};

	const baseClasses = `m3-state inline-flex items-center justify-center gap-2 transition-colors border rounded-full cursor-pointer ${disabledClasses} focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2`;

	// Farb-Utilities des Aufrufers ersetzen die der Variante, statt mit ihnen zu konkurrieren:
	// Tailwind-Utilities haben alle dieselbe Spezifität, also entschiede sonst die Reihenfolge
	// im Stylesheet und nicht die im class-Attribut. Nur Basisfarben werden ersetzt;
	// Zustandsvarianten (hover:, disabled:, focus-within:) bleiben stehen, und Größenangaben
	// (text-label-small, text-label-large) gelten nicht als Farbe.
	const FARBE =
		/^(bg|border|ring|text)-(transparent|primary|secondary|error|success|warning|surface|outline|scrim|inverse-|on-)/;
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
	class="{baseClasses} {variant === 'symbol'
		? symbolSizes[size]
		: sizes[size]} {ohneFlaecheGesperrt[variant] ?? flaecheGesperrt} {variantClasses} {className}"
	{...rest}
>
	{@render children?.()}
</button>
