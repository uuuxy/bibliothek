<script>
	/** @type {{
	 *   children?: any,
	 *   variant?: 'primary' | 'secondary' | 'danger' | 'success' | 'danger-solid' | 'ghost',
	 *   size?: 'sm' | 'md' | 'lg',
	 *   class?: string,
	 *   [key: string]: any
	 * }} */
	let { children, variant = 'primary', size = 'md', class: className = '', ...rest } = $props();

	// Ohne hover:bg-*: Die Rückmeldung kommt jetzt aus dem State-Layer (.m3-state,
	// siehe app.css) — eine Schicht in der Textfarbe über der UNVERÄNDERTEN Fläche,
	// wie Material 3 es vorsieht. Vorher legte jede Variante ihren eigenen
	// Hover-Farbwechsel fest; das musste sechsmal gepflegt werden und passte bei
	// getönten Flächen (danger, success) ohnehin nie ganz.
	const variants = {
		primary: 'bg-blue-600 text-white border-transparent shadow-sm',
		secondary: 'bg-white border-slate-200 text-slate-700 shadow-sm',
		danger: 'bg-rose-50 border-rose-200 text-rose-700 shadow-sm',
		'danger-solid': 'bg-rose-600 text-white border-transparent shadow-sm',
		success: 'bg-emerald-50 border-emerald-200 text-emerald-700 shadow-sm',
		ghost: 'bg-transparent border-transparent text-slate-700'
	};

	// Feste Höhen statt reinem Padding: Nur so stehen Buttons neben Eingabefeldern und
	// in Tabellenzeilen auf einer Linie. Gemessen wurden vorher NEUN verschiedene
	// Button-Höhen zwischen 15 und 38 px — jede Komponente brachte ihre eigene mit.
	//
	// md = 36 px ist die gemeinsame Control-Höhe der Anwendung: Suchfelder, Selects und
	// Buttons müssen denselben Wert tragen, sonst steht in jeder Werkzeugleiste ein
	// Feld neben einem Button auf zwei verschiedenen Grundlinien.
	const sizes = {
		sm: 'h-7 px-2.5 text-xs',
		md: 'h-9 px-3 text-sm',
		lg: 'h-10 px-4 text-sm'
	};

	// rounded-full: In Material 3 IST der Button eine Pille — das ist keine
	// Verspieltheit, sondern die Rollen-Zuordnung der M3-Shape-Skala (Menüs 4 px,
	// Chips 8, Karten 12, Dialoge 28, Buttons voll). Die frühere Entscheidung
	// gegen die Pille stammt aus der Zeit vor dem M3-Ziel und ist am 04.08.2026
	// ausdrücklich zurückgenommen worden.
	const baseClasses =
		'm3-state inline-flex items-center justify-center gap-2 font-semibold transition-colors border rounded-full cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2';

	// Farb-Utilities des Aufrufers ERSETZEN die der Variante, statt mit ihnen zu konkurrieren.
	//
	// Ohne das gewinnt nicht der Aufrufer, sondern der Zufall: Tailwind-Utilities haben alle
	// dieselbe Spezifität, also entscheidet die Reihenfolge IM STYLESHEET — nicht die im
	// class-Attribut. `bg-white` der Variante steht hinter `bg-blue-50` des Aufrufers, also
	// blieb ein getönter Button stumm weiß. Gemessen an sechs Stellen (Medienkatalog-Toolbar,
	// Klassenkarte, Stammdaten, Buch-Akte, Offline-Banner, Mahnwesen), alle unbemerkt.
	//
	// Nur Basisfarben werden ersetzt; Zustandsvarianten (hover:, disabled:, focus-within:)
	// bleiben stehen, und Größenangaben (text-label-small, text-sm) gelten nicht als Farbe.
	const FARBE =
		/^(bg|border|ring|text)-(slate|gray|zinc|blue|indigo|emerald|green|amber|orange|rose|red|white|black|transparent)/;
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

<button class="{baseClasses} {sizes[size]} {variantClasses} {className}" {...rest}>
	{@render children?.()}
</button>
