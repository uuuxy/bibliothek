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
	// KEIN Schatten, in keiner Variante — nachgezaehlt in der M3-Token-Spezifikation
	// (material-web v0.192, "Design system display name: Google Material 3"):
	//
	//   filled-button    container-elevation level0   (= 0dp)   kein Rahmen
	//   outlined-button  KEIN elevation-Token         outline-width 1px
	//   text-button      KEIN elevation-Token         kein Rahmen
	//   elevated-button  container-elevation level1   kein Rahmen
	//
	// Genau EINE der vier Bauformen ist erhoben — und das ist die, die dafuer eine
	// eigene Variante hat und KEINEN Rahmen traegt. Ueber alle 84 Bauteile der
	// Spezifikation gilt das ausnahmslos: 6 tragen einen Rahmen, 36 eine Erhebung
	// ueber level0, die Schnittmenge ist LEER. Rahmen und Erhebung sind in M3 zwei
	// Wege, eine Flaeche abzugrenzen — nie eine Summe.
	//
	// Hier standen sie als Summe: `baseClasses` gibt jeder Variante einen `border`,
	// fuenf setzten zusaetzlich `shadow-sm`. Im Browser gemessen (15 Routen,
	// Ruhezustand) waren das 224 sichtbare Vorkommen — jeder Knopf der Anwendung
	// ausser `ghost`.
	const variants = {
		primary: 'bg-blue-600 text-white border-transparent',
		secondary: 'bg-white border-slate-200 text-slate-700',
		danger: 'bg-rose-50 border-rose-200 text-rose-700',
		'danger-solid': 'bg-rose-600 text-white border-transparent',
		success: 'bg-emerald-50 border-emerald-200 text-emerald-700',
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
		// sm war 28/12 — M3 kennt keinen Knopf unter label-large 14 (Audit Schritt 3).
		sm: 'h-8 px-2.5 text-sm',
		md: 'h-9 px-3 text-sm',
		lg: 'h-10 px-4 text-sm'
	};

	// rounded-full: In Material 3 IST der Button eine Pille — das ist keine
	// Verspieltheit, sondern die Rollen-Zuordnung der M3-Shape-Skala (Menüs 4 px,
	// Chips 8, Karten 12, Dialoge 28, Buttons voll). Die frühere Entscheidung
	// gegen die Pille stammt aus der Zeit vor dem M3-Ziel und ist am 04.08.2026
	// ausdrücklich zurückgenommen worden.
	// Deaktiviert: EIN Zustand für alle Varianten, nach Material 3 — Fläche in der
	// Textfarbe bei 12 %, Beschriftung bei 38 %. Kein Rahmen, kein Schatten: Ein
	// abgeschalteter Knopf soll flach wirken, nicht wie ein bedienbarer mit Schleier.
	//
	// Vorher stand hier `disabled:opacity-50`, und weil das nur blass macht statt
	// umzufärben, hat sich jede Ansicht ihren eigenen Zustand gebaut. Gemessen am
	// 07.08.2026: 21 Fundstellen mit VIER Opazitäten (40/50/60/100) und DREI
	// Flächenfarben. Sichtbar wurde es als Farbunterschied zwischen zwei Bildschirmen —
	// "Anlegen" (Signaturen) war blaues Primary bei 50 %, "A4-Bogen drucken"
	// (Druck-Center) grau bei voller Deckkraft. Zwei Knöpfe, derselbe Zustand, zwei
	// Aussagen.
	//
	// !-Präfix, weil die Varianten-Klassen (bg-blue-600 …) dieselbe Spezifität haben und
	// bei gleicher Spezifität die Stylesheet-Reihenfolge entscheidet, nicht die im
	// class-Attribut — dieselbe Falle wie bei den Farb-Overrides weiter unten.
	const disabledClasses =
		'disabled:cursor-not-allowed disabled:!bg-on-surface/[0.12] disabled:!text-on-surface/[0.38] disabled:!border-transparent disabled:!shadow-none';

	const baseClasses = `m3-state inline-flex items-center justify-center gap-2 font-semibold transition-colors border rounded-full cursor-pointer ${disabledClasses} focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2`;

	// Farb-Utilities des Aufrufers ERSETZEN die der Variante, statt mit ihnen zu konkurrieren.
	//
	// Ohne das gewinnt nicht der Aufrufer, sondern der Zufall: Tailwind-Utilities haben alle
	// dieselbe Spezifität, also entscheidet die Reihenfolge IM STYLESHEET — nicht die im
	// class-Attribut. `bg-white` der Variante steht hinter `bg-blue-50` des Aufrufers, also
	// blieb ein getönter Button stumm weiß. Gemessen an sechs Stellen (Medienkatalog-Toolbar,
	// Klassenkarte, Stammdaten, Buch-Akte, Offline-Banner, Mahnwesen), alle unbemerkt.
	//
	// Seit 02.09.2026 auch die M3-Rollen (text-error, bg-primary …): Sonst blieb `text-error` am
	// Ghost-Knopf stumm, weil `text-slate-700` der Variante im Stylesheet dahinter steht.
	// Nur Basisfarben werden ersetzt; Zustandsvarianten (hover:, disabled:, focus-within:)
	// bleiben stehen, und Größenangaben (text-label-small, text-sm) gelten nicht als Farbe.
	const FARBE =
		/^(bg|border|ring|text)-(slate|gray|zinc|blue|indigo|emerald|green|amber|orange|rose|red|white|black|transparent|primary|secondary|tertiary|error|surface|outline|on-)/;
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
