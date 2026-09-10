<script module>
	// Läuft einmal je Modul, nicht je Instanz — die Nummern bleiben eindeutig.
	let zaehler = 0;
</script>

<script>
	/**
	 * @component Feld
	 * DAS Eingabefeld der Anwendung — Material 3, Variante „outlined".
	 *
	 * Bis zum 25.08.2026 gab es drei Wörter für dieselbe Sache: `SettingField`, `InputField`
	 * und 81 handgebaute `<input>` in 47 Dateien mit sieben Radien, vier Fokusfarben und
	 * drei Flächen — dieselbe Geschichte wie bei den zehn Suchfeld-Kopien.
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
	 *   - Rahmen `outline` (die M3-Rolle des outlined text field im Ruhezustand), Fläche
	 *     `surface-container-lowest`, im Fokus `primary` mit 1-px-Ring: exakt das Rezept
	 *     von Select.svelte, damit Feld und Auswahlfeld in einer Zeile als EIN Vokabular
	 *     lesen. Bis zum 09.09.2026 stand hier `outline-variant` — auf Weiß 1,7:1, und
	 *     WCAG 1.4.11 verlangt 3:1 für den Rand eines Bedienelements; `outline` hat 4,5:1.
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
	 * @prop {Snippet} [vorlaufend] - Inhalt links IM Feld (Symbol, Präfix-Text wie „Gültig bis 31.07.").
	 * @prop {Snippet} [nachlaufend] - Inhalt rechts IM Feld (Einheit „Stück", Knöpfe, Spinner).
	 * @prop {boolean} [mehrzeilig=false] - Mehrzeiliges Textfeld (M3: text field, multiline).
	 * @prop {number} [zeilen=3] - Sichtbare Zeilen im mehrzeiligen Feld.
	 *
	 * Mehrzeilig ist dieselbe Bauform, nur höher — ohne die 36-px-Höhe, die für eine Zeile
	 * gilt. Ein zweites Bauteil dafür wäre der Anfang derselben Geschichte wie bei den 81
	 * handgebauten Feldern; das Höhen-Gate misst input und select, textarea nicht.
	 *
	 * vor-/nachlaufend liegen als Überlagerung über dem Feld (wie in Suchfeld.svelte); das
	 * Feld bleibt das gerahmte 36-px-Element und bekommt pl-10 bzw. pr-10. Wer breiteren
	 * Inhalt legt, gibt die Innenabstände über `feld` mit (z. B. feld="pl-36").
	 */

	/** @type {{ value?: any, label?: string, vorlaufend?: import('svelte').Snippet, nachlaufend?: import('svelte').Snippet, type?: 'text'|'number'|'email'|'date'|'month'|'password'|'search'|'tel'|'url', hint?: string, ungueltig?: boolean, class?: string, feld?: string, element?: HTMLInputElement, id?: string, mehrzeilig?: boolean, zeilen?: number } & Omit<import('svelte/elements').HTMLInputAttributes, 'value'|'type'|'class'|'id'>} */
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
		vorlaufend = undefined,
		nachlaufend = undefined,
		mehrzeilig = false,
		zeilen = 3,
		...rest
	} = $props();

	// Eigene Nummer je Feld: Der Hilfetext braucht ein Ziel für aria-describedby, und
	// zwei Felder derselben Seite dürfen sich keine ID teilen.
	const nummer = ++zaehler;
	const feldId = $derived(id ?? `feld-${nummer}`);
	const hinweisId = $derived(`${feldId}-hinweis`);

	const inputClass = $derived(
		// Breite NUR setzen, wenn `feld` keine mitbringt: w-full und w-64 sind gleich
		// spezifisch, dann entschiede die Stylesheet-Reihenfolge statt des Aufrufs
		// (Tailwind-Kaskaden-Falle, am Ausweis-Feld gesehen: w-64 verlor gegen w-full).
		// Mehrzeilig wächst mit den Zeilen, die 36-px-Grundlinie gilt für eine Zeile.
		(mehrzeilig ? 'w-full py-2 leading-relaxed ' : /\bw-/.test(feld) ? 'h-9 ' : 'h-9 w-full ') +
			'rounded-xl border bg-surface-container-lowest px-3 text-sm text-on-surface ' +
			'transition-colors placeholder:text-outline focus:outline-none focus:ring-1 ' +
			'disabled:cursor-not-allowed disabled:opacity-40 read-only:text-on-surface-variant ' +
			(ungueltig
				? 'border-error focus:border-error focus:ring-error '
				: 'border-outline focus:border-primary focus:ring-primary ') +
			(vorlaufend && !/\bpl-/.test(feld) ? 'pl-10 ' : '') +
			(nachlaufend && !/\bpr-/.test(feld) ? 'pr-10 ' : '') +
			feld
	);
	const beschreibung = $derived(hint ? hinweisId : undefined);
	// $derived, nicht const: `rest` ist reaktiv, eine Konstante fror den Anfangswert ein.
	const restFuerTextarea = $derived(
		/** @type {import('svelte/elements').HTMLTextareaAttributes} */ (/** @type {any} */ (rest))
	);
</script>

{#snippet roh()}
	{#if mehrzeilig}
		<!-- `restFuerTextarea` ist die Naht zwischen zwei Elementtypen: `rest` ist als
		     HTMLInputAttributes typisiert (das Bauteil ist zuerst ein <input>). Die
		     Attribute, die hier durchgereicht werden, gibt es an beiden Elementen. -->
		<textarea
			id={feldId}
			rows={zeilen}
			aria-describedby={beschreibung}
			aria-invalid={ungueltig || undefined}
			bind:value
			class={inputClass}
			{...restFuerTextarea}></textarea>
	{:else if type === 'number'}
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

{#snippet eingabe()}
	{#if vorlaufend || nachlaufend}
		<div class="relative">
			{#if vorlaufend}
				<div
					class="pointer-events-none absolute top-1/2 left-3 flex -translate-y-1/2 items-center gap-2 text-sm whitespace-nowrap text-on-surface-variant"
				>
					{@render vorlaufend()}
				</div>
			{/if}
			{@render roh()}
			{#if nachlaufend}
				<div
					class="absolute top-1/2 right-3 flex -translate-y-1/2 items-center gap-1 text-sm text-on-surface-variant"
				>
					{@render nachlaufend()}
				</div>
			{/if}
		</div>
	{:else}
		{@render roh()}
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
