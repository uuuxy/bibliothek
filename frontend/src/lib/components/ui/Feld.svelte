<script module>
	// Läuft einmal je Modul, nicht je Instanz — die Nummern bleiben eindeutig.
	let zaehler = 0;
</script>

<script>
	/**
	 * @component Feld
	 * DAS Eingabefeld der Anwendung — Material 3, Variante „outlined".
	 *
	 * Bis zum 25.08.2026 gab es drei Wörter für dieselbe Sache: `SettingField`
	 * (Einstellungen, M3), `InputField` (ein Nutzer, Slate-Optik) und 81 handgebaute
	 * `<input>` in 47 Dateien mit sieben Radien, vier Fokusfarben und drei Flächen —
	 * kopiert und dann auseinandergelaufen, dieselbe Geschichte wie bei den zehn
	 * Suchfeld-Kopien (Suchfeld.svelte). Buttons, Auswahlfelder und Suchfelder waren
	 * längst normiert; die Textfelder trugen nur die 36-px-Höhe aus styles/basis.css.
	 *
	 * Die Form ist entschieden, nicht gewählt:
	 *   - Höhe 36 px wie jedes Bedienelement (basis.css, Gate e2e/control-hoehen.spec.js).
	 *     M3 nennt 56 dp; das gilt für Formulare am Telefon, nicht für ein
	 *     Verwaltungswerkzeug, dessen Knöpfe seit dem 08.08. auf 36 px stehen.
	 *   - Radius `rounded-xl`: „Karten und Eingabefelder → 12 px" (55a1d4b0, 07.08.2026).
	 *     SettingField stand auf rounded-sm und war damit die Ausnahme neben dem
	 *     Select in derselben Zeile.
	 *   - Schrift `text-sm` wie Select und Button size="md" — ein Feld neben einem
	 *     Auswahlfeld darf nicht größer schreiben als dieses.
	 *   - Rahmen `outline-variant`, Fläche `surface-container-lowest`, im Fokus
	 *     `primary` mit 1-px-Ring: exakt das Rezept von Select.svelte, damit Feld und
	 *     Auswahlfeld in einer Zeile als EIN Vokabular lesen.
	 *   - Outlined und nicht filled, weil die Arbeitsfläche weiß ist (f2320e1) — eine
	 *     getönte Feldfläche wäre der einzige graue Block auf der Seite.
	 *
	 * Beschriftung ÜBER dem Feld statt schwebend: Ein schwebendes Label wandert beim
	 * Tippen weg und ist im Ruhezustand vom Platzhalter nicht zu unterscheiden.
	 *
	 * Mit `label` liegen die drei Zeilen (Beschriftung, Feld, Hinweis) als SUBGRID im
	 * Raster des Aufrufers — sonst rutscht ein Feld eine Zeile tiefer, sobald seine
	 * Beschriftung umbricht (flasch3, 23.08.2026). Wer ein Feld über mehrere Spalten
	 * zieht, gibt `class="sm:col-span-2"` HIER mit und packt es NICHT in ein <div>.
	 *
	 * OHNE `label` (Tabellenzelle, Werkzeugleiste) ist das Bauteil nur das Feld — dann
	 * ist `aria-label` Pflicht, sonst hat der Screenreader einen namenlosen Kasten.
	 *
	 * Der Hilfetext hängt an aria-describedby und liegt AUSSERHALB des <label>: Ein
	 * Name benennt, eine Beschreibung erklärt.
	 *
	 * Zwei-Wege-Bindung erfordert eine Komponente (Snippets können `bind:` nicht
	 * zurückpropagieren). Alles, was hier nicht benannt ist (placeholder, min, max,
	 * step, maxlength, pattern, required, disabled, readonly, autocomplete, inputmode,
	 * list, aria-label, oninput, onchange, onkeydown, onfocus, onblur …), landet
	 * unverändert auf dem <input>.
	 *
	 * @prop {string|number} [value] - Gebundener Wert (bindable).
	 * @prop {string} [label] - Beschriftung über dem Feld.
	 * @prop {'text'|'number'|'email'|'date'|'month'|'password'|'search'|'tel'|'url'} [type='text']
	 * @prop {string} [hint=''] - Hilfetext unter dem Feld.
	 * @prop {boolean} [ungueltig=false] - Fehlerzustand: Rahmen und Hinweis in `error`.
	 * @prop {string} [class] - Rasterangaben des Aufrufers, z. B. "sm:col-span-2" (nur mit label).
	 * @prop {string} [feld] - Zusatzklassen NUR fürs <input>, z. B. "w-20 text-center".
	 * @prop {HTMLInputElement} [element] - bind:this-Ersatz (bindable).
	 */

	/** @type {{ value?: any, label?: string, type?: 'text'|'number'|'email'|'date'|'month'|'password'|'search'|'tel'|'url', hint?: string, ungueltig?: boolean, class?: string, feld?: string, element?: HTMLInputElement, id?: string } & Omit<import('svelte/elements').HTMLInputAttributes, 'value'|'type'|'class'|'id'>} */
	let {
		value = $bindable(),
		label = undefined,
		type = 'text',
		hint = '',
		ungueltig = false,
		class: className = '',
		feld = '',
		element = $bindable(),
		id = undefined,
		...rest
	} = $props();

	// Eigene Nummer je Feld: Der Hilfetext braucht ein Ziel für aria-describedby, und
	// zwei Felder derselben Seite dürfen sich keine ID teilen.
	const nummer = ++zaehler;
	const feldId = $derived(id ?? `feld-${nummer}`);
	const hinweisId = $derived(`${feldId}-hinweis`);

	const inputClass = $derived(
		'h-9 w-full rounded-xl border bg-surface-container-lowest px-3 text-sm text-on-surface ' +
			'transition-colors placeholder:text-outline focus:outline-none focus:ring-1 ' +
			'disabled:cursor-not-allowed disabled:opacity-40 read-only:text-on-surface-variant ' +
			(ungueltig
				? 'border-error focus:border-error focus:ring-error '
				: 'border-outline-variant focus:border-primary focus:ring-primary ') +
			feld
	);
	const beschreibung = $derived(hint ? hinweisId : undefined);
</script>

{#snippet eingabe()}
	{#if type === 'number'}
		<input
			bind:this={element}
			id={feldId}
			type="number"
			aria-describedby={beschreibung}
			aria-invalid={ungueltig || undefined}
			bind:value
			class={inputClass}
			{...rest}
		/>
	{:else}
		<input
			bind:this={element}
			id={feldId}
			{type}
			aria-describedby={beschreibung}
			aria-invalid={ungueltig || undefined}
			bind:value
			class={inputClass}
			{...rest}
		/>
	{/if}
{/snippet}

{#if label}
	<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5 {className}">
		<label for={feldId} class="text-sm font-medium text-on-surface-variant">{label}</label>
		{@render eingabe()}
		{#if hint}
			<span id={hinweisId} class="text-xs {ungueltig ? 'text-error' : 'text-on-surface-variant'}"
				>{hint}</span
			>
		{/if}
	</div>
{:else}
	{@render eingabe()}
	{#if hint}
		<span id={hinweisId} class="text-xs {ungueltig ? 'text-error' : 'text-on-surface-variant'}"
			>{hint}</span
		>
	{/if}
{/if}
