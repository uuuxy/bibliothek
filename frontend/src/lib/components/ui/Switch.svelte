<!-- @component Switch — der Ein/Aus-Schalter der Anwendung (Material 3).

     Material 3 unterscheidet: Ein Häkchen wählt aus einer Liste aus, ein Schalter legt einen
     Zustand um. „Händler beklebt die Bücher" und „Benutzerkonto ist aktiv" sind Zustände —
     also Schalter.

     Maße nach M3: Spur 52×32 dp, Griff 16 dp im Aus-Zustand und 24 dp im Ein-Zustand. Das
     Wachsen des Griffs macht den Zustand auch ohne Farbe erkennbar.

     Farben nach der Token-Datei (material-web v0.192, switch): an Spur primary mit Griff
     on-primary; aus Spur surface-container-highest mit Rand und Griff in outline.

     Der Aufrufer muss eine Beschriftung mitgeben — sichtbar über `id` und ein eigenes
     <label>, oder per `label` als aria-label. -->
<script>
	/** @type {{
	 *   checked: boolean,
	 *   label?: string,
	 *   id?: string,
	 *   disabled?: boolean,
	 *   onchange?: (v: boolean) => void
	 * }} */
	let { checked = $bindable(false), label = '', id, disabled = false, onchange } = $props();

	function umlegen() {
		if (disabled) return;
		checked = !checked;
		onchange?.(checked);
	}
</script>

<button
	{id}
	type="button"
	role="switch"
	aria-checked={checked}
	aria-label={label || undefined}
	{disabled}
	onclick={umlegen}
	class="relative inline-flex h-8 w-13 shrink-0 items-center rounded-full border-2 transition-colors duration-200 focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 {checked
		? 'border-primary bg-primary'
		: 'border-outline bg-surface-container-highest'} {disabled ? '' : 'cursor-pointer'}"
>
	<!-- Der Griff wächst beim Einschalten von 16 auf 24 px (M3). Damit ist der Zustand
	     auch ohne Farbunterschied zu erkennen. -->
	<span
		class="pointer-events-none inline-block transform rounded-full shadow-sm transition-all duration-200 ease-in-out {checked
			? 'ml-1 h-6 w-6 translate-x-5 bg-on-primary'
			: 'ml-1.5 h-4 w-4 translate-x-0 bg-outline'}"
	></span>
</button>
