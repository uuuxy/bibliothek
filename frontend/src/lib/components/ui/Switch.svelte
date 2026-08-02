<!-- @component Switch — der Ein/Aus-Schalter der Anwendung (Material 3).

     Vorher gab es DREI Darstellungen für dieselbe Ja/Nein-Entscheidung: einen
     role="switch"-Button in den Systemeinstellungen (32×56 px, grün), einen
     peer-checked-Nachbau in der Benutzerverwaltung (24×40 px, blau) und ein einfaches
     Häkchen in der Lieferantenverwaltung. Drei Formen für einen Gedanken — man erkennt
     nicht wieder, was man schon kennt.

     Material 3 unterscheidet bewusst: Ein Häkchen wählt aus einer Liste aus, ein Schalter
     legt einen Zustand um. „Händler beklebt die Bücher" und „Benutzerkonto ist aktiv" sind
     Zustände, keine Auswahl — also Schalter.

     Maße nach M3: Spur 52×32 dp, Griff 16 dp im Aus-Zustand und 24 dp im Ein-Zustand. Das
     Wachsen des Griffs ist kein Zierrat, es macht den Zustand auch ohne Farbe erkennbar —
     wichtig bei Rot-Grün-Sehschwäche.

     Farbe ist blue-600 wie bei Button variant="primary": M3 färbt die aktive Spur mit der
     Primärfarbe. Der Schalter in den Systemeinstellungen war als einziger grün.

     Der Aufrufer MUSS eine Beschriftung mitgeben — entweder sichtbar über `id` und ein
     eigenes <label>, oder per `label` als aria-label. Der Nachbau in der Benutzer-
     verwaltung hatte gar keinen zugänglichen Namen: Der Screenreader las dort nur
     „Kontrollkästchen", die danebenstehende Erklärung gehörte niemandem. -->
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
	class="relative inline-flex h-8 w-13 shrink-0 items-center rounded-full border-2 transition-colors duration-200 focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 {checked
		? 'border-blue-600 bg-blue-600'
		: 'border-slate-300 bg-slate-200'} {disabled ? '' : 'cursor-pointer'}"
>
	<!-- Der Griff wächst beim Einschalten von 16 auf 24 px (M3). Damit ist der Zustand
	     auch ohne Farbunterschied zu erkennen. -->
	<span
		class="pointer-events-none inline-block transform rounded-full shadow-sm transition-all duration-200 ease-in-out {checked
			? 'ml-1 h-6 w-6 translate-x-5 bg-white'
			: 'ml-1.5 h-4 w-4 translate-x-0 bg-slate-500'}"
	></span>
</button>
