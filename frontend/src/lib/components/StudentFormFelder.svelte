<!-- @component StudentFormFelder — die Eingabefelder für einen neuen Schüler.

     Die Klasse kommt normalerweise aus den Lesergruppen; „Manuell eingeben" schaltet
     auf ein Freitextfeld um, weil es Klassen gibt, die noch nicht als Gruppe
     angelegt sind. Der Weg zurück steht als Knopf IM Feld — sonst säße man in der
     Freitexteingabe fest.

     Das Geburtsdatum ist Pflicht — nicht als Stammdatum, sondern als SCHLÜSSEL: Der
     LUSD-Export der Schule hat keine Schüler-ID; der Import erkennt einen von Hand
     angelegten Schüler nur über Name + Geburtsdatum wieder. Ohne Datum entstünde beim
     nächsten Import ein Duplikat. Das Backend lehnt es ebenfalls ab (zwei Türen). -->
<script>
	import Select from './ui/Select.svelte';

	/**
	 * @type {{
	 *   vorname: string, nachname: string, geburtsdatum: string,
	 *   klasse: string, barcode: string,
	 *   freieKlasse: boolean, readerGroups: any[]
	 * }}
	 */
	let {
		vorname = $bindable(),
		nachname = $bindable(),
		geburtsdatum = $bindable(),
		klasse = $bindable(),
		barcode = $bindable(),
		freieKlasse = $bindable(),
		readerGroups = []
	} = $props();

	const klassenOptionen = $derived([
		...readerGroups.map((/** @type {any} */ g) => ({
			value: g.kuerzel,
			label: `${g.kuerzel} (${g.bezeichnung})`
		})),
		{ value: '__custom__', label: 'Manuell eingeben…' }
	]);
</script>

<label class="block text-xs font-medium text-slate-400"
	>Vorname *
	<input
		type="text"
		bind:value={vorname}
		placeholder="z.B. Max"
		class="mt-1.5 w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-sans"
	/>
</label>

<label class="block text-xs font-medium text-slate-400"
	>Nachname *
	<input
		type="text"
		bind:value={nachname}
		placeholder="z.B. Mustermann"
		class="mt-1.5 w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-sans"
	/>
</label>

<label class="block text-xs font-medium text-slate-400"
	>Geburtsdatum *
	<input
		type="date"
		required
		bind:value={geburtsdatum}
		class="mt-1.5 w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-sans"
	/>
	<span class="mt-1 block text-label-small font-normal text-on-surface-variant"
		>Pflicht: Der LUSD-Import erkennt den Schüler nur über Name + Geburtsdatum wieder — ohne Datum
		würde er beim nächsten Import doppelt angelegt.</span
	>
</label>

<div class="block text-xs font-medium text-slate-400">
	<label for="schueler-klasse">Klasse *</label>
	<div class="mt-1.5 flex gap-2">
		{#if !freieKlasse}
			<Select
				id="schueler-klasse"
				bind:value={klasse}
				options={klassenOptionen}
				placeholder="Lesergruppe / Klasse auswählen"
				onchange={(wert) => {
					if (wert === '__custom__') {
						freieKlasse = true;
						klasse = '';
					}
				}}
			/>
		{:else}
			<div class="relative w-full">
				<input
					type="text"
					bind:value={klasse}
					placeholder="z.B. 10b"
					class="w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-sans"
				/>
				<button
					type="button"
					onclick={() => {
						freieKlasse = false;
						klasse = '';
					}}
					class="absolute right-2.5 top-1/2 -translate-y-1/2 text-xs font-semibold text-blue-600 hover:text-blue-700 transition-colors bg-transparent border-none cursor-pointer"
					>Auswahl</button
				>
			</div>
		{/if}
	</div>
</div>

<label class="block text-xs font-medium text-slate-400"
	>Barcode-ID (optional)
	<input
		type="text"
		bind:value={barcode}
		placeholder="Wird automatisch generiert, wenn leer"
		class="mt-1.5 w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all"
	/>
</label>
