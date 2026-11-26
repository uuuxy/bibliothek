<!--
  @component Datumsfeld
  Ein Datum eintippen oder im Kalender wählen, M3 Docked date picker: „Dates can be added by using
  a keyboard or by navigating the calendar UI; both options are immediately available“. Der Wert
  ist das Datum als YYYY-MM-DD wie beim Feld type="date" des Browsers; angezeigt wird TT.MM.JJJJ.

  Getippt wird ohne Eingabemaske; ordentlich geschrieben wird das Datum nach Enter oder beim
  Verlassen, M3 (Date pickers, Accessibility): „Don't automatically format the date by adding
  slashes or other special characters while the user is typing“. Der Kalender geht über den Knopf
  im Feld auf, „The calendar icon is the exclusive entry point for the date picker“, oder mit
  Alt+↓ wie in Googles Angular Material.
-->
<script>
	import Feld from './Feld.svelte';
	import Icon from './Icon.svelte';
	import Kalender from './Kalender.svelte';
	import { folgeLage } from './lageFolgen.js';
	import { lageUnter } from './selectGeometrie.js';
	import { leseDatum } from '../../utils/datum.js';
	import { formatDatum } from '../../utils/format.js';

	// Maße des Kalenders: 360 px breit wie der Docked date picker, Kopf, Wochentage, sechs Wochen
	// zu 48 px und die Knöpfe darunter.
	const BREITE = 360;
	const HOEHE = 456;

	/**
	 * @type {{ value?: string | null, hint?: string, ungueltig?: boolean, disabled?: boolean } &
	 *   Omit<import('svelte').ComponentProps<typeof Feld>, 'value' | 'type' | 'hint' | 'ungueltig' | 'disabled' | 'nachlaufend' | 'element'>}
	 */
	let {
		value = $bindable(),
		hint = '',
		ungueltig = undefined,
		disabled = false,
		oninput = undefined,
		...rest
	} = $props();

	const kennung = $props.id();
	let text = $state(formatDatum(value));
	let zuletzt = value;
	let falsch = $state(false);
	let offen = $state(false);
	/** @type {HTMLInputElement | undefined} */
	let element = $state();
	/** @type {HTMLButtonElement | undefined} */
	let knopf = $state();
	let box = $state({ left: 0, top: 0, breite: BREITE });

	// Unter dem Feld steht das Format, M3 (Date pickers, Accessibility): „The helper text … should
	// specify the date format“. Ein Fehler ersetzt es, M3 (Text fields): „replace supporting text
	// with error text“; der Satz zum falsch getippten Datum ist Googles eigener aus Compose.
	const unterzeile = $derived.by(() => {
		if (falsch) return 'Datum entspricht nicht dem erwarteten Format: TT.MM.JJJJ';
		if (ungueltig && hint) return hint;
		return hint ? `TT.MM.JJJJ · ${hint}` : 'TT.MM.JJJJ';
	});

	// Ein Wert von außen (Laden, Zurücksetzen) erscheint ordentlich geschrieben; einer aus dem
	// Tippen nicht, sonst spränge die Schreibmarke.
	$effect(() => {
		if (value === zuletzt) return;
		zuletzt = value;
		text = formatDatum(value);
		falsch = false;
	});

	$effect(() => {
		if (!offen) return;
		return folgeLage(
			() => element,
			() => {
				if (element) box = lageUnter(element, HOEHE, BREITE);
			}
		);
	});

	/** @param {Event & { currentTarget: HTMLInputElement }} e */
	function getippt(e) {
		text = e.currentTarget.value;
		// Wie beim Feld des Browsers: ein unvollständiges Datum ist noch keins.
		value = zuletzt = leseDatum(text) ?? '';
		// Wer das Feld einsetzt, hört das Tippen mit, auch ein unvollständiges Datum.
		oninput?.(e);
	}

	function festlegen() {
		const iso = leseDatum(text);
		falsch = iso === null;
		if (iso) text = formatDatum(iso);
	}

	/** @param {KeyboardEvent} e */
	function taste(e) {
		if (e.key === 'Enter') festlegen();
		else if (e.altKey && e.key === 'ArrowDown') {
			e.preventDefault();
			offen = !disabled;
		}
	}

	/** @param {boolean} zurueck */
	function schliessen(zurueck) {
		offen = false;
		if (zurueck) knopf?.focus();
	}

	/** @param {string} iso */
	function uebernehmen(iso) {
		if (iso) {
			value = zuletzt = iso;
			text = formatDatum(iso);
			falsch = false;
		}
		schliessen(true);
	}
</script>

<Feld
	bind:element
	value={text}
	type="text"
	inputmode="numeric"
	autocomplete="off"
	hint={unterzeile}
	ungueltig={falsch || ungueltig}
	aria-haspopup="dialog"
	{disabled}
	oninput={getippt}
	onkeydown={taste}
	onblur={festlegen}
	{...rest}
>
	{#snippet nachlaufend()}
		<button
			bind:this={knopf}
			type="button"
			class="icon-btn text-on-surface-variant"
			aria-label="Kalender öffnen"
			data-tip="Kalender öffnen: Alt+↓"
			aria-haspopup="dialog"
			aria-expanded={offen}
			aria-controls={offen ? `${kennung}-kalender` : undefined}
			{disabled}
			onclick={() => (offen ? schliessen(true) : (offen = true))}
		>
			<Icon name="calendar_today" groesse={24} />
		</button>
	{/snippet}
</Feld>

{#if offen}
	<Kalender
		wert={value || ''}
		{box}
		kennung="{kennung}-kalender"
		ausloeser={knopf}
		onuebernehmen={uebernehmen}
		onschliessen={schliessen}
	/>
{/if}
