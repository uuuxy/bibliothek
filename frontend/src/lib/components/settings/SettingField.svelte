<script module>
	// Läuft einmal je Modul, nicht je Instanz — die Nummern bleiben eindeutig.
	let zaehler = 0;
</script>

<script>
	/**
	 * @component SettingField
	 * Das Eingabefeld der Einstellungen — Material 3, Variante „outlined".
	 *
	 * Bis zum 23.08.2026 war es ein Unterstrich-Feld: kein Container, nur eine Linie
	 * unter dem Text. Das ist Material 2. M3 kennt für Textfelder genau zwei Formen,
	 * `filled` (getönte Fläche) und `outlined` (Rahmen) — der Unterstrich wurde mit M3
	 * abgeschafft, weil eine einzelne Linie den Klickbereich nicht zeigt: Man sieht
	 * nicht, wo das Feld anfängt, nur wo es aufhört.
	 *
	 * Outlined und nicht filled, weil die Arbeitsfläche dieser Anwendung weiß ist
	 * (f2320e1) — eine getönte Feldfläche wäre hier der einzige graue Block auf der
	 * Seite. Der Rahmen kommt aus `outline-variant`, im Fokus aus `primary`.
	 *
	 * Beschriftung ÜBER dem Feld statt schwebend: Ein schwebendes Label wandert beim
	 * Tippen weg und ist im Ruhezustand vom Platzhalter nicht zu unterscheiden — in
	 * einem Formular, das man einmal im Jahr ausfüllt, ist das die falsche Sparsamkeit.
	 *
	 * Höhe 36 px wie jedes andere Bedienelement (styles/basis.css, Gate
	 * e2e/control-hoehen.spec.js). M3 nennt für Textfelder 56 dp; das gilt für
	 * Formulare am Telefon, nicht für ein Verwaltungswerkzeug, dessen Knöpfe und
	 * Auswahlfelder seit dem 08.08. geschlossen auf 36 px stehen.
	 *
	 * Zwei-Wege-Bindung erfordert eine Komponente (Snippets können `bind:` nicht
	 * zurückpropagieren), daher hier statt eines {#snippet} gekapselt.
	 *
	 * Der Hilfetext hängt an aria-describedby und liegt AUSSERHALB des <label>. Vorher
	 * stand er darin, und damit hieß das Feld für einen Screenreader „Öffentliche
	 * Adresse Leer = keine Bestätigungs-Links verschicken" — Name und Erläuterung in
	 * einem Atemzug. Ein Name benennt, eine Beschreibung erklärt.
	 *
	 * @prop {string|number} value - Gebundener Wert (bindable).
	 * @prop {string} label - Feldbeschriftung.
	 * @prop {'number'|'text'|'email'|'date'} [type='number'] - Eingabetyp.
	 * @prop {string} [hint=''] - Optionaler Hilfetext unter dem Feld.
	 * @prop {number} [min] - Minimalwert (number).
	 * @prop {number} [max] - Maximalwert (number).
	 * @prop {string} [placeholder=''] - Platzhaltertext.
	 * @prop {string} [pattern] - Validierungs-Pattern (text).
	 * @prop {number} [maxlength] - Maximale Zeichenanzahl (text).
	 */

	/** @type {{ value: string|number, label: string, type?: 'number'|'text'|'email'|'date', hint?: string, min?: number, max?: number, placeholder?: string, pattern?: string, maxlength?: number }} */
	let {
		value = $bindable(),
		label,
		type = 'number',
		hint = '',
		min,
		max,
		placeholder = '',
		pattern,
		maxlength
	} = $props();

	// Eigene Nummer je Feld: Der Hilfetext braucht ein Ziel für aria-describedby, und
	// zwei Felder derselben Seite dürfen sich keine ID teilen.
	const feldId = `setting-${++zaehler}`;
	const hinweisId = `${feldId}-hinweis`;

	const inputClass =
		'w-full rounded-sm border border-outline-variant bg-surface-container-lowest px-3 ' +
		'text-base text-on-surface transition-colors placeholder:text-outline ' +
		'focus:border-primary focus:outline-none';
</script>

<div class="flex flex-col gap-1.5">
	<label for={feldId} class="text-sm font-medium text-on-surface-variant">{label}</label>
	{#if type === 'number'}
		<input
			id={feldId}
			type="number"
			{min}
			{max}
			{placeholder}
			aria-describedby={hint ? hinweisId : undefined}
			bind:value
			class={inputClass}
		/>
	{:else}
		<input
			id={feldId}
			{type}
			{placeholder}
			{pattern}
			{maxlength}
			aria-describedby={hint ? hinweisId : undefined}
			bind:value
			class={inputClass}
		/>
	{/if}
	{#if hint}
		<span id={hinweisId} class="text-xs text-on-surface-variant">{hint}</span>
	{/if}
</div>
