<!-- @component Switch — der Ein/Aus-Schalter der Anwendung (Material 3).

     Material 3 unterscheidet: Ein Häkchen wählt aus einer Liste aus, ein Schalter legt einen
     Zustand um. „Händler beklebt die Bücher" und „Benutzerkonto ist aktiv" sind Zustände —
     also Schalter.

     Maße nach M3 (Switch, Specs): Spur 52×32 dp, Griff 16 dp im Aus-Zustand und 24 dp im
     Ein-Zustand, Zustandsschicht 40 dp, Trefferfläche 48 dp („Target Size 48dp"). Das Wachsen
     des Griffs macht den Zustand auch ohne Farbe erkennbar.

     Farben nach der Token-Datei (material-web, switch): an Spur primary mit Griff on-primary;
     aus Spur surface-container-highest mit Rand und Griff in outline. Die Zustandsschicht ist
     ein Kreis um den Griff, an in primary, aus in on-surface („selected.hover.state-layer.color:
     primary", „unselected.hover.state-layer.color: on-surface"), nicht die Fläche des ganzen
     Knopfs wie bei den übrigen Knöpfen (no-state-layer).

     Der Aufrufer muss eine Beschriftung mitgeben — sichtbar über `id` und ein eigenes
     <label>, oder per `label` als aria-label. -->
<script>
	/** @type {{
	 *   checked: boolean,
	 *   etikett?: string,
	 *   id?: string,
	 *   disabled?: boolean,
	 *   onchange?: (v: boolean) => void
	 * }} */
	let { checked = $bindable(false), etikett = '', id, disabled = false, onchange } = $props();

	function umlegen() {
		if (disabled) return;
		checked = !checked;
		onchange?.(checked);
	}
</script>

<!-- before: die Trefferfläche, 52 × 48 px bis an die Außenkante des Rands (die Einrückung zählt
     vom Inneren des 2-px-Rands); sie ragt über den Knopf hinaus und verschiebt nichts, wie
     .icon-btn in styles/komponenten.css. -->
<button
	{id}
	type="button"
	role="switch"
	aria-checked={checked}
	aria-label={etikett || undefined}
	{disabled}
	onclick={umlegen}
	class="group no-state-layer relative inline-flex h-8 w-13 shrink-0 items-center rounded-full border-2 transition-colors duration-200 before:absolute before:-inset-x-0.5 before:-inset-y-2.5 focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 {checked
		? 'border-primary bg-primary'
		: 'border-outline bg-surface-container-highest'} {disabled ? '' : 'cursor-pointer'}"
>
	<!-- Der Griff wächst beim Einschalten von 16 auf 24 px (M3). Damit ist der Zustand
	     auch ohne Farbunterschied zu erkennen. Der Kreis der Zustandsschicht liegt hinter ihm
	     und wandert mit. -->
	<span
		class="pointer-events-none relative inline-block transition-all duration-200 ease-in-out {checked
			? 'ml-1 h-6 w-6 translate-x-5'
			: 'ml-1.5 h-4 w-4 translate-x-0'}"
	>
		<span
			class="absolute top-1/2 left-1/2 size-10 -translate-x-1/2 -translate-y-1/2 rounded-full opacity-0 transition-opacity group-hover:opacity-8 group-focus-visible:opacity-10 group-active:opacity-10 group-disabled:hidden {checked
				? 'bg-primary'
				: 'bg-on-surface'}"
		></span>
		<span
			class="relative block h-full w-full rounded-full shadow-sm {checked
				? 'bg-on-primary'
				: 'bg-outline'}"
		></span>
	</span>
</button>
