<!-- @component Die Gruppe „An der Schule" der Titelmaske: was die Schule über das Buch
     entscheidet. Die Wahl Bibliothek oder Lernmittel steht oben, weil die Pflicht zur
     Signatur, der Schulzweig, das Mehrjahresband und die anderen Auflagen von ihr abhängen. -->
<script>
	import { bisNachVon, mehrjahresbandHinweis } from '$lib/components/admin/buch_form_optionen.js';
	import Select from '../../../../lib/components/ui/Select.svelte';
	import Feld from '../../../../lib/components/ui/Feld.svelte';
	import Segmente from '../../../../lib/components/ui/Segmente.svelte';
	import Kaestchen from '../../../../lib/components/ui/Kaestchen.svelte';
	import SignaturFeld from './SignaturFeld.svelte';
	import BuchAuflagen from './BuchAuflagen.svelte';

	let { formular = $bindable(), systematikListe = [] } = $props();

	const ARTEN = [
		{ wert: 'bibliothek', text: 'Bibliothek' },
		{ wert: 'lernmittel', text: 'Lernmittel' }
	];
	const faecher = $derived([
		{ value: '', label: 'Kein Fach' },
		...systematikListe.map((/** @type {any} */ s) => ({
			value: s.bezeichnung,
			label: `${s.kuerzel} - ${s.bezeichnung}`
		}))
	]);
	// Der Schulzweig steht nur am Lernmittel: Der Portal-Reiter der Schulbücher filtert nach
	// ihm, über ein Bibliotheksbuch sagt er nichts.
	const ZWEIGE = [
		{ value: '', label: 'Alle Zweige' },
		...['Gymnasium', 'Realschule', 'Hauptschule', 'Förderstufe', 'Oberstufe'].map((z) => ({
			value: z,
			label: z
		}))
	];
	const BESCHRIFTUNG = 'text-sm font-medium text-on-surface-variant';

	/** „bis" geht mit „von" mit, solange es keinen eigenen Wert trägt (bisNachVon).
	 * @param {number|string|null|undefined} von */
	function setzeVon(von) {
		formular.jahrgangBis = bisNachVon(formular.jahrgangVon, von, formular.jahrgangBis);
		formular.jahrgangVon = von;
	}

	/** @param {string} art */
	function waehleArt(art) {
		formular.istLernmittel = art === 'lernmittel';
		// Der Server weist ein Mehrjahresband am Bibliotheksbuch ab.
		if (!formular.istLernmittel) formular.mehrjahresband = false;
	}
</script>

<div class="mt-8 border-t border-outline-variant pt-6">
	<h3 class="text-lg font-semibold text-on-surface">An der Schule</h3>
	<div class="mt-4 space-y-5">
		<!-- Zwei Möglichkeiten, die einander ausschließen, sind eine Auswahl und kein Schalter:
		     Ein Schalter wirkt sofort, diese Angabe gilt erst mit „Speichern". -->
		<div>
			<Segmente
				etikett="Art des Buchs"
				optionen={ARTEN}
				wert={formular.istLernmittel ? 'lernmittel' : 'bibliothek'}
				onwahl={waehleArt}
			/>
			<p class="mt-1.5 text-xs text-on-surface-variant">
				Lernmittel: Schulbuch, das die Schule fürs Schuljahr leiht. Frist bis zum Stichtag, zählt
				nicht ins Ausleihlimit, erscheint nicht im öffentlichen Katalog.
			</p>
		</div>

		<!-- Der Standort steht am Exemplar und wird in der Buchakte geändert, nicht hier. -->
		<div class="grid grid-cols-2 gap-4">
			<SignaturFeld bind:formular />
		</div>

		<!-- Auswahlfelder wie ui/Feld mit Beschriftung: drei Zeilen im Subgrid, damit die
		     Nachbarn einer Zeile fluchten. -->
		<div class="grid grid-cols-2 gap-4">
			<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5">
				<label for="buch-fach" class={BESCHRIFTUNG}>Fach</label>
				<Select
					id="buch-fach"
					bind:value={formular.subject}
					options={faecher}
					placeholder="Fach auswählen"
				/>
			</div>
			{#if formular.istLernmittel}
				<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5">
					<label for="buch-schulzweig" class={BESCHRIFTUNG}>Schulzweig</label>
					<Select
						id="buch-schulzweig"
						bind:value={formular.track}
						options={ZWEIGE}
						aria-describedby="buch-schulzweig-hinweis"
					/>
					<span id="buch-schulzweig-hinweis" class="text-xs text-on-surface-variant">
						Nur setzen, wenn das Buch wirklich einem Zweig gehört — leer heißt „gilt für alle".
					</span>
				</div>
			{/if}
		</div>

		<!-- Bei einem Lernmittel ist die Spanne der Unterricht, kein Lesealter, und bei einem
		     Mehrjahresband die Laufzeit beim Kind: Die Zahl kommt aus „bis". -->
		<div class="grid grid-cols-2 gap-4">
			<Feld
				id="buch-jahrgang-von"
				label={formular.istLernmittel ? 'Im Unterricht von Jahrgang' : 'Geeignet für Jahrgang von'}
				type="number"
				min="1"
				max="13"
				bind:value={() => formular.jahrgangVon, setzeVon}
			/>
			<Feld
				id="buch-jahrgang-bis"
				label="bis Jahrgang"
				type="number"
				min="1"
				max="13"
				bind:value={formular.jahrgangBis}
			/>
		</div>

		{#if formular.istLernmittel}
			<!-- Als Spalte gesetzt: In einer Textzeile höbe die Trefferfläche des Kästchens
			     (40 px) die Zeile an, und der Hinweis rückte ab. -->
			<div class="flex flex-col items-start">
				<Kaestchen
					id="buch-mehrjahresband"
					bind:checked={formular.mehrjahresband}
					label="Mehrjahresband"
					aria-describedby="buch-mehrjahresband-hinweis"
				/>
				<p id="buch-mehrjahresband-hinweis" class="mt-1.5 pl-7.5 text-xs text-on-surface-variant">
					{mehrjahresbandHinweis(
						formular.mehrjahresband,
						formular.jahrgangVon,
						formular.jahrgangBis
					)}
				</p>
			</div>
		{/if}

		<!-- Zuordnen und Lösen brauchen einen gespeicherten Titel. -->
		{#if formular.id}
			<BuchAuflagen {formular} />
		{/if}
	</div>
</div>
