<!-- @component StudentFormFelder — die Eingabefelder für einen neuen Schüler.

     Die Klasse wird aus den Klassen der Schule gewählt (GET /api/klassen), nicht getippt
     (docs/OFFEN.md 5.18, 30.09.2026): „Manuell eingeben…" legte bei einem Tippfehler eine
     Klasse an, die es an der Schule nicht gibt. Eine ganz neue Klasse kommt mit dem
     LUSD-Abgleich. Ließen sich die Klassen nicht laden, sagt das der Fehlertext statt des
     Hinweises auf den LUSD-Abgleich (klassenVorschlaege.svelte.js).

     Das Geburtsdatum ist Pflicht — nicht als Stammdatum, sondern als SCHLÜSSEL: Der
     LUSD-Export der Schule hat keine Schüler-ID; der Import erkennt einen von Hand
     angelegten Schüler nur über Name + Geburtsdatum wieder. Ohne Datum entstünde beim
     nächsten Import ein Duplikat. Das Backend lehnt es ebenfalls ab (zwei Türen). -->
<script>
	import Select from './ui/Select.svelte';
	import Feld from './ui/Feld.svelte';
	import { KLASSEN_LADEFEHLER, klassenPlatzhalter } from './students/klassenVorschlaege.svelte.js';

	/**
	 * @type {{
	 *   vorname: string, nachname: string, geburtsdatum: string,
	 *   klasse: string, barcode: string,
	 *   klassen: string[], klassenFehler?: boolean
	 * }}
	 */
	let {
		vorname = $bindable(),
		nachname = $bindable(),
		geburtsdatum = $bindable(),
		klasse = $bindable(),
		barcode = $bindable(),
		klassen = [],
		klassenFehler = false
	} = $props();

	const klassenOptionen = $derived(klassen.map((k) => ({ value: k, label: k })));
</script>

<Feld label="Vorname *" bind:value={vorname} placeholder="z.B. Max" />

<Feld label="Nachname *" bind:value={nachname} placeholder="z.B. Mustermann" />

<Feld
	label="Geburtsdatum *"
	type="date"
	required
	bind:value={geburtsdatum}
	hint="Pflicht: Der LUSD-Import erkennt den Schüler nur über Name + Geburtsdatum wieder — ohne Datum würde er beim nächsten Import doppelt angelegt."
/>

<div class="grid gap-y-1.5">
	<label for="schueler-klasse" class="text-sm font-medium text-on-surface-variant">Klasse *</label>
	<Select
		id="schueler-klasse"
		bind:value={klasse}
		options={klassenOptionen}
		placeholder={klassenPlatzhalter(klassen.length, klassenFehler)}
	/>
	{#if klassenFehler}
		<span class="text-xs text-error">{KLASSEN_LADEFEHLER}</span>
	{:else if !klassen.length}
		<span class="text-xs text-on-surface-variant">Die Klassen kommen mit dem LUSD-Abgleich.</span>
	{/if}
</div>

<Feld
	label="Barcode-ID (optional)"
	bind:value={barcode}
	placeholder="Wird automatisch generiert, wenn leer"
/>
