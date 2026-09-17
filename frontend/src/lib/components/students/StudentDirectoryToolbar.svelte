<script>
	import { Plus } from '@lucide/svelte';
	import Suchpille from '../ui/Suchpille.svelte';
	import Button from '../ui/Button.svelte';
	import Select from '../ui/Select.svelte';

	/**
	 * jahrgang: '' = alle. Das Auswahlfeld steht seit dem 17.09.2026 hier — das
	 * Sichtungsprotokoll des Medienzentrums vom 16.09.2026 vermisste eine „Sortier- oder
	 * Filteroption nach Klassen bzw. Jahrgängen" und präzisierte unter Anpassungswünschen:
	 * „Gemeint ist jedoch eine Auswahl nach Jahrgängen."
	 *
	 * Gefiltert wird auf dem SERVER, wie gesucht: Die ungefilterte Liste ist bei 500 Zeilen
	 * gekappt, und ein Filter im Browser säße hinter dieser Kappung — er zeigte dann einen
	 * Teil des Jahrgangs und sähe dabei vollständig aus.
	 *
	 * @type {{ searchQuery?: string, jahrgang?: string, jahrgaenge?: number[], jahrgaengeFehler?: boolean, darfAnlegen?: boolean, trefferzahl?: number, suchend?: boolean, gekuerzt?: boolean, onsearch?: () => void, oncreate?: () => void }}
	 */
	let {
		searchQuery = $bindable(''),
		jahrgang = $bindable(''),
		jahrgaenge = [],
		jahrgaengeFehler = false,
		darfAnlegen = false,
		trefferzahl = 0,
		suchend = false,
		gekuerzt = false,
		onsearch,
		oncreate
	} = $props();

	// Bei einem Ladefehler benennt das Feld seinen Zustand SELBST, statt „Alle Jahrgänge"
	// zu zeigen: Ein Platzhalter hilft hier nicht, weil bei value='' die gewählte Option
	// angezeigt wird — und die läse sich wie eine heile Liste ohne Inhalt.
	const jahrgangsOptionen = $derived(
		jahrgaengeFehler
			? [{ value: '', label: 'Jahrgänge nicht geladen' }]
			: [
					{ value: '', label: 'Alle Jahrgänge' },
					...jahrgaenge.map((j) => ({ value: String(j), label: `Jahrgang ${j}` }))
				]
	);
</script>

<!-- Flach und edge-to-edge: kein Kachel-Container, nur dezenter Abstand zu den Tabs. -->
<div class="mt-4 flex flex-col gap-3">
	<Suchpille
		id="schuelerdatei-suchfeld"
		bind:wert={searchQuery}
		oninput={onsearch}
		platzhalter="Name, Klasse oder Ausweisnummer eingeben …"
		etikett="Leser suchen"
	/>

	<div class="flex items-center gap-4">
		<!-- 36 px wie jedes Bedienelement; die Breite steht hier, weil `class` in Select die
		     Standardbreite w-full ersetzt (sonst zöge das Feld die ganze Zeile). -->
		<Select
			id="leserdatei-jahrgang"
			bind:value={jahrgang}
			options={jahrgangsOptionen}
			onchange={onsearch}
			disabled={jahrgaengeFehler}
			class="w-44"
			aria-label="Nach Jahrgang filtern"
		/>

		{#if darfAnlegen}
			<Button variant="primary" onclick={oncreate} aria-label="Neuen Leser anlegen">
				<Plus class="w-4 h-4" />
				Neuer Leser
			</Button>
		{/if}

		<!-- Sagt, was tatsächlich zu sehen ist. Vorher stand hier "500 / 500", während die
		     Schule 875 Schüler hatte — die Zahl bestätigte dem Benutzer eine Vollständigkeit,
		     die es nicht gab, und machte das Fehlen einzelner Namen unerklärlich.

		     Die Kappung steht ZUERST, seit dem 17.09.2026. Vorher gewann `suchend`, und ein
		     Jahrgangsfilter zählt dazu: Eine bei 500 abgeschnittene Jahrgangsliste meldete
		     „Treffer: 500" — dieselbe falsche Vollständigkeit wie damals, nur eine Ebene
		     tiefer. Was gekappt ist, muss es sagen, egal warum die Liste eingegrenzt wurde. -->
		<div class="ml-auto shrink-0 text-xs font-semibold text-slate-500">
			{#if gekuerzt}
				Erste {trefferzahl} — zum Finden bitte suchen
			{:else if suchend}
				Treffer: {trefferzahl}
			{:else}
				Einträge: {trefferzahl}
			{/if}
		</div>
	</div>
</div>
