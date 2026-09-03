<script>
	/**
	 * @component PortalSchulbuecher
	 * Schulbücher für die Fachsprecher (Peter, 03.09.2026): „Englisch möchte nur die
	 * Englisch-Bücher … im Grunde ähnlich wie bei Klassensätzen. Wir brauchen alle
	 * Fachbereiche, eine Suchfunktion und Filterfunktionen."
	 *
	 * Also dieselbe Bauform wie unter Bibliothek → Klassensätze: oben Suche und Filter,
	 * darunter eine aufklappbare Karte je Fach mit den Cover-Kacheln (FachKarte). Zwei
	 * verworfene Fassungen desselben Tages stehen im git log: eine Kachelwand je Fach und
	 * eine Chip-Leiste — beide zeigten dem Fachsprecher immer die ganze Schule.
	 *
	 * Gefiltert wird SERVERSEITIG (Jahrgang, Schulzweig, Suchtext): Sonst zeigten die
	 * Zahlen am Fach und die Excel-Datei etwas anderes als die Liste darunter.
	 * Grundlage ist allein der Lernmittel-Schalter; Daten über die Portal-Tür
	 * /api/portal/lernmittel (Anmeldung genügt, kein view_books).
	 */
	import { onMount } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import { apiFetch } from '../../apiFetch.js';
	import Feld from '../ui/Feld.svelte';
	import Select from '../ui/Select.svelte';
	import FachKarte from './FachKarte.svelte';

	/** @type {{ fach: string, titel: number, gesamt: number, verliehen: number, verfuegbar: number }[]} */
	let faecher = $state.raw([]);
	/** @type {any[]} */
	let titel = $state.raw([]);
	let suche = $state('');
	let jahrgang = $state(0);
	let zweig = $state('');
	/** aufgeklappte Fächer ('' = ohne Fach) */
	const offen = new SvelteSet();
	let laedt = $state(true);
	let fehler = $state('');
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let timer;

	const JAHRGAENGE = [
		{ value: 0, label: 'Alle Jahrgänge' },
		...[5, 6, 7, 8, 9, 10, 11, 12, 13].map((j) => ({ value: j, label: `Jahrgang ${j}` }))
	];
	// „-" ist der Filterwert für „kein Zweig gesetzt" (inventur.ZweigOhne) — die meisten
	// Altbestände tragen keinen, weil Littera den Zweig nie erfasst hat.
	const ZWEIGE = [
		{ value: '', label: 'Alle Schulzweige' },
		...['Gymnasium', 'Realschule', 'Hauptschule', 'Förderstufe', 'Oberstufe'].map((z) => ({
			value: z,
			label: z
		})),
		{ value: '-', label: 'Ohne Schulzweig' }
	];

	const parameter = $derived(
		new URLSearchParams(
			/** @type {Record<string,string>} */ (
				Object.fromEntries(
					[
						['q', suche.trim()],
						['jahrgang', jahrgang ? String(jahrgang) : ''],
						['zweig', zweig]
					].filter(([, v]) => v)
				)
			)
		).toString()
	);
	// Der Export ist ein PDF mit Coverbildern (Peter, 03.09.2026: „es rechnet niemand,
	// also können wir Excel löschen") und trägt dieselbe Filterung wie die Ansicht.
	/** @param {string} fach */
	const exportUrl = (fach) =>
		`/api/portal/lernmittel/export?fach=${encodeURIComponent(fach)}${parameter ? `&${parameter}` : ''}`;

	async function lade() {
		laedt = true;
		fehler = '';
		try {
			const res = await apiFetch(`/api/portal/lernmittel${parameter ? `?${parameter}` : ''}`);
			if (!res.ok) {
				fehler = 'Schulbücher konnten nicht geladen werden.';
				return;
			}
			const data = await res.json();
			faecher = data.faecher ?? [];
			titel = data.titel ?? [];
		} catch {
			fehler = 'Schulbücher konnten nicht geladen werden.';
		} finally {
			laedt = false;
		}
	}

	onMount(lade);

	// Tippen entprellt, Auswahl sofort — wie in den übrigen Listen des Hauses.
	function tippen() {
		clearTimeout(timer);
		timer = setTimeout(lade, 250);
	}

	/** @param {string} fach */
	function klappe(fach) {
		if (!offen.delete(fach)) offen.add(fach);
	}

	/** @param {string} fach */
	const buecherDesFachs = (fach) => titel.filter((b) => (b.subject ?? '') === fach);
</script>

<div class="flex flex-col gap-4 pt-2">
	<div class="flex flex-wrap items-center gap-3">
		<Feld
			id="schulbuecher-suche"
			type="search"
			bind:value={suche}
			oninput={tippen}
			placeholder="Titel, ISBN, Autor oder Fach"
			aria-label="Schulbücher durchsuchen"
			feld="w-full sm:w-72"
		/>
		<Select
			bind:value={jahrgang}
			options={JAHRGAENGE}
			onchange={lade}
			class="w-44"
			aria-label="Nach Jahrgang filtern"
		/>
		<Select
			bind:value={zweig}
			options={ZWEIGE}
			onchange={lade}
			class="w-48"
			aria-label="Nach Schulzweig filtern"
		/>
	</div>

	{#if fehler}
		<p class="text-sm font-bold text-error">{fehler}</p>
	{:else if laedt && faecher.length === 0}
		<p class="py-4 text-sm text-on-surface-variant">Lädt …</p>
	{:else if faecher.length === 0}
		<p class="py-4 text-sm text-on-surface-variant">
			{suche.trim() || jahrgang || zweig
				? 'Keine Schulbücher passen zu dieser Auswahl.'
				: 'Noch keine Schulbücher markiert — der Lernmittel-Schalter am Titel entscheidet.'}
		</p>
	{:else}
		<div data-testid="schulbuecher-faecher">
			{#each faecher as f (f.fach)}
				<FachKarte
					fach={f}
					buecher={buecherDesFachs(f.fach)}
					offen={offen.has(f.fach)}
					exportUrl={exportUrl(f.fach)}
					onToggle={() => klappe(f.fach)}
				/>
			{/each}
		</div>
	{/if}
</div>
