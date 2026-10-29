<!-- @component Switch — der Ein/Aus-Schalter der Anwendung (Material 3).

     Material 3 unterscheidet: Ein Häkchen wählt aus einer Liste aus, ein Schalter legt einen
     Zustand um. „Händler beklebt die Bücher" und „Benutzerkonto ist aktiv" sind Zustände —
     also Schalter.

     Maße nach M3 (Switch, Specs): Spur 52×32 dp, Griff 16 dp aus, 24 dp an und 28 dp beim
     Drücken, Zustandsschicht 40 dp, Trefferfläche 48 dp („Target Size 48dp"). Der Griff sitzt
     mittig in einem 32-dp-Feld am Anfang oder Ende der Spur (material-web, _handle.scss) und
     wächst deshalb um seine Mitte. Das Wachsen macht den Zustand auch ohne Farbe erkennbar.

     Farben nach der Token-Datei (material-web, md-comp-switch): an Spur primary, Griff on-primary
     und beim Überfahren, Fokus und Druck primary-container; aus Spur surface-container-highest
     mit Rand in outline, Griff outline und sonst on-surface-variant. Die Zustandsschicht ist ein
     Kreis um den Griff, an in primary, aus in on-surface, nicht die Fläche des ganzen Knopfs wie
     bei den übrigen Knöpfen (no-state-layer). Der Griff hat keinen Schatten: „handle.elevation"
     ist dort veraltet.

     Der Aufrufer muss eine Beschriftung mitgeben — sichtbar über `id` und ein eigenes
     <label>, oder per `etikett` als aria-label. -->
<script>
	/** @type {{
	 *   checked: boolean,
	 *   etikett?: string,
	 *   id?: string,
	 *   disabled?: boolean,
	 *   onchange?: (v: boolean) => void
	 * }} */
	let { checked = $bindable(false), etikett = '', id, disabled = false, onchange } = $props();

	// Gesperrt nach den Token „disabled": Spur 12 %, Griff an in surface, aus in on-surface 38 %.
	// Die Zustände stehen nur am bedienbaren Schalter, weil ein gesperrter Knopf :hover weiter meldet.
	const spur = $derived(
		disabled
			? checked
				? 'border-transparent bg-on-surface/12'
				: 'border-on-surface/12 bg-surface-container-highest/12'
			: checked
				? 'border-primary bg-primary'
				: 'border-outline bg-surface-container-highest'
	);
	const griff = $derived(
		disabled
			? checked
				? 'size-6 bg-surface'
				: 'size-4 bg-on-surface/38'
			: checked
				? 'size-6 bg-on-primary group-hover:bg-primary-container group-focus-visible:bg-primary-container group-active:size-7 group-active:bg-primary-container'
				: 'size-4 bg-outline group-hover:bg-on-surface-variant group-focus-visible:bg-on-surface-variant group-active:size-7 group-active:bg-on-surface-variant'
	);

	function umlegen() {
		if (disabled) return;
		checked = !checked;
		onchange?.(checked);
	}
</script>

<!-- before: die Trefferfläche, 52 × 48 px bis an die Außenkante des Rands (die Einrückung zählt
     vom Inneren des 2-px-Rands); sie ragt über den Knopf hinaus und verschiebt nichts, wie
     .icon-btn in styles/komponenten.css. Das Feld des Griffs beginnt aus demselben Grund
     bei -0.5. -->
<button
	{id}
	type="button"
	role="switch"
	aria-checked={checked}
	aria-label={etikett || undefined}
	{disabled}
	onclick={umlegen}
	class="group no-state-layer relative inline-flex h-8 w-13 shrink-0 rounded-full border-2 transition-colors duration-200 before:absolute before:-inset-x-0.5 before:-inset-y-2.5 focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed {spur} {disabled
		? ''
		: 'cursor-pointer'}"
>
	<span
		class="pointer-events-none absolute -top-0.5 -left-0.5 flex size-8 items-center justify-center transition-transform duration-200 ease-in-out {checked
			? 'translate-x-5'
			: ''}"
	>
		<span
			class="absolute top-1/2 left-1/2 size-10 -translate-x-1/2 -translate-y-1/2 rounded-full opacity-0 transition-opacity group-hover:opacity-8 group-focus-visible:opacity-10 group-active:opacity-10 group-disabled:hidden {checked
				? 'bg-primary'
				: 'bg-on-surface'}"
		></span>
		<span data-griff class="relative rounded-full transition-all duration-200 ease-in-out {griff}"
		></span>
	</span>
</button>
