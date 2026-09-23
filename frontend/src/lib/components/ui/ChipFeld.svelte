<script module>
	// Läuft einmal je Modul, nicht je Instanz — die Listen-IDs bleiben eindeutig.
	let zaehler = 0;
</script>

<script>
	/**
	 * @component ChipFeld
	 * Ein Eingabefeld für MEHRERE Werte: Jeder übernommene Wert wird ein Input-Chip mit
	 * einem × zum Entfernen. Material 3, Chips › Input chips: „They enable user input and
	 * verify that input by converting text into chips"; das Schließen-Symbol ist dort
	 * „required and must be used to remove the chip".
	 *
	 * Gebaut für die Schlagworte am Titel (Migration 138). Keins der vorhandenen Bauteile
	 * nimmt mehrere freie Werte: Feld nimmt einen, Select einen aus fester Liste, und
	 * LmfKlasseChip ist ein einzelner Chip ohne Eingabe. Deshalb setzt sich dieses Bauteil
	 * aus den beiden zusammen, statt sie nachzubauen — das Feld IST ui/Feld (36 px, derselbe
	 * Rahmen), die Chips haben die Form von LmfKlasseChip (32 px, Radius 8 px,
	 * secondary-container, × auf 32 × 32 px wie das Gate icon-trefferflaechen verlangt).
	 *
	 * Übernommen wird mit Enter, mit Komma, mit der Auswahl eines Vorschlags und beim
	 * Verlassen des Feldes: Ein getipptes, nicht bestätigtes Wort geht beim Speichern nicht
	 * verloren. Kommas trennen auch beim Einfügen („Krimi, Freundschaft").
	 *
	 * Doppelte zählen ohne Rücksicht auf Groß- und Kleinschreibung, und ein Vorschlag gibt
	 * seine Schreibweise vor: Wer „fantasy" tippt, bekommt den Chip „Fantasy" — dieselbe
	 * Regel, die der Server beim Speichern anwendet (repository.SetzeSchlagworte).
	 *
	 * Die Chips stehen UNTER dem Feld und brechen um (M3, Placement: „In a stacked list";
	 * „Input chips can wrap to a new row if all chips need to be visible").
	 *
	 * @prop {string[] | null} [werte] - Die gewählten Werte (bindable); null = nicht geladen.
	 * @prop {{ wert: string, beschreibung?: string }[]} [vorschlaege] - Angebot beim Tippen.
	 * @prop {number} [max=30] - Höchstzahl; darüber meldet das Feld den Fehlerzustand.
	 * @prop {number} [maxZeichen=80] - Länge eines Werts.
	 * @prop {string[]} [angebote] - Werte zum Anklicken unter den Chips (M3 Suggestion chips),
	 *   etwa der Schlagwort-Vorschlag aus der DNB. Ein Klick übernimmt; bis dahin ist nichts
	 *   eingetragen. Was schon gewählt ist, steht nicht mehr darunter.
	 * @prop {string} [angeboteEtikett] - Name dieser Zeile, sichtbar und für Screenreader.
	 */
	import { Plus, X } from '@lucide/svelte';
	import Feld from './Feld.svelte';

	/** @type {{ werte?: string[] | null, id?: string, label?: string, hint?: string, placeholder?: string, vorschlaege?: { wert: string, beschreibung?: string }[], max?: number, maxZeichen?: number, disabled?: boolean, angebote?: string[], angeboteEtikett?: string, 'aria-label'?: string }} */
	let {
		werte = $bindable([]),
		id = undefined,
		label = undefined,
		hint = 'Mit Enter oder Komma übernehmen.',
		placeholder = '',
		vorschlaege = [],
		max = 30,
		maxZeichen = 80,
		disabled = false,
		angebote = [],
		angeboteEtikett = 'Vorschläge',
		'aria-label': ariaLabel = undefined
	} = $props();

	const nummer = ++zaehler;
	const feldId = $derived(id ?? `chipfeld-${nummer}`);
	const listeId = $derived(`${feldId}-vorschlaege`);

	// null heißt beim Aufrufer „nicht geladen" (die Katalogliste liefert Schlagworte so).
	// Angezeigt wird dann nichts; ob das Feld offen ist, entscheidet der Aufrufer.
	const liste = $derived(werte ?? []);
	let eingabe = $state('');
	let voll = $state(false);
	/** @type {HTMLInputElement | undefined} */
	let feldElement = $state();

	/** @param {string} wert */
	const schluessel = (wert) => wert.toLowerCase();

	/** Übernimmt, was im Feld steht — Kommas trennen mehrere Werte. */
	function uebernimm() {
		for (const teil of eingabe.split(',')) {
			const wert = teil.split(/\s+/).filter(Boolean).join(' ');
			if (!wert || liste.some((w) => schluessel(w) === schluessel(wert))) continue;
			if (liste.length >= max) {
				voll = true;
				break;
			}
			const vorschlag = vorschlaege.find((v) => schluessel(v.wert) === schluessel(wert));
			werte = [...liste, vorschlag ? vorschlag.wert : wert];
		}
		eingabe = '';
	}

	/** @param {KeyboardEvent} e */
	function taste(e) {
		if (e.key === 'Enter' || e.key === ',') {
			e.preventDefault();
			uebernimm();
		}
	}

	/** Auswahl aus der Vorschlagsliste: Der Browser meldet sie als Ersetzung, nicht als
	 *  getippten Text. Ein Klick auf den Vorschlag ist damit schon die Übernahme.
	 *  @param {Event} e */
	function eingegeben(e) {
		const art = /** @type {InputEvent} */ (e).inputType;
		if (art === 'insertReplacementText' || art === undefined) uebernimm();
	}

	/** Die Angebote, die noch nicht gewählt sind. */
	const offeneAngebote = $derived(
		angebote.filter((a) => !liste.some((w) => schluessel(w) === schluessel(a)))
	);

	/** Übernimmt ein Angebot — mit derselben Obergrenze wie getippter Text.
	 *  @param {string} wert */
	function nimm(wert) {
		if (liste.length >= max) {
			voll = true;
			return;
		}
		werte = [...liste, wert];
	}

	/** @param {string} wert */
	function entferne(wert) {
		werte = liste.filter((w) => w !== wert);
		voll = false;
		// Der Knopf verschwindet mit dem Chip; ohne das stünde der Fokus auf der Seite.
		feldElement?.focus();
	}
</script>

<div class="space-y-2">
	<Feld
		id={feldId}
		{label}
		aria-label={label ? undefined : ariaLabel}
		bind:value={eingabe}
		bind:element={feldElement}
		list={listeId}
		{placeholder}
		{disabled}
		maxlength={maxZeichen}
		autocomplete="off"
		ungueltig={voll}
		hint={voll ? `Höchstens ${max} — zuerst einen entfernen.` : hint}
		onkeydown={taste}
		oninput={eingegeben}
		onblur={uebernimm}
	/>
	<datalist id={listeId}>
		{#each vorschlaege as v (v.wert)}
			<option value={v.wert}>{v.beschreibung ? `${v.wert} — ${v.beschreibung}` : v.wert}</option>
		{/each}
	</datalist>
	{#if liste.length}
		<!-- Eigener Name, nicht der des Feldes: Zwei Elemente mit demselben Namen sagt ein
		     Screenreader als dasselbe an. -->
		<ul class="flex flex-wrap gap-2" aria-label="{label ?? ariaLabel} (gewählt)">
			{#each liste as wert (schluessel(wert))}
				<li
					class="inline-flex h-8 items-center gap-1 rounded-md bg-secondary-container pl-3 text-sm font-medium text-on-secondary-container"
				>
					{wert}
					<button
						type="button"
						onclick={() => entferne(wert)}
						{disabled}
						class="flex h-8 w-8 cursor-pointer items-center justify-center rounded-full hover:bg-on-secondary-container/10 disabled:cursor-not-allowed disabled:opacity-40"
						title="„{wert}“ entfernen"
						aria-label="„{wert}“ entfernen"
					>
						<X class="h-4 w-4" aria-hidden="true" />
					</button>
				</li>
			{/each}
		</ul>
	{/if}
	{#if offeneAngebote.length}
		<!-- M3 Suggestion chips (material-web, _md-comp-suggestion-chip.scss): 32 px, Ecke 8 px,
		     Umriss outline, Text on-surface-variant, Symbol vorn in primary — „Suggestion chips
		     help narrow a user's intent by presenting dynamically generated suggestions". Die
		     Beschriftung steht in eigener Zeile: In schmalen Spalten (Bestellfenster) brach sie
		     sonst neben dem ersten Chip um, und die übrigen standen versetzt darunter. -->
		<p class="text-xs text-on-surface-variant" aria-hidden="true">{angeboteEtikett}</p>
		<div class="flex flex-wrap gap-2" role="group" aria-label={angeboteEtikett}>
			{#each offeneAngebote as wert (schluessel(wert))}
				<button
					type="button"
					onclick={() => nimm(wert)}
					{disabled}
					aria-label="„{wert}“ übernehmen"
					class="flex h-8 cursor-pointer items-center gap-2 rounded-md border border-outline pr-4 pl-2 text-sm font-medium text-on-surface-variant disabled:cursor-not-allowed disabled:opacity-40"
				>
					<Plus class="h-4.5 w-4.5 shrink-0 text-primary" aria-hidden="true" />
					{wert}
				</button>
			{/each}
		</div>
	{/if}
</div>
