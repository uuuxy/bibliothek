<!-- @component Kategorisierung eines Titels: Lernmittel-Schalter, Fach, Klasse, Jahrgangsspanne.

     Der Schalter ersetzt seit dem 02.09.2026 (Migration 093) die Auswahl „Schulzweig"
     plus das Textpräfix „LMF" in der Signatur. Ob ein Buch ein Lernmittel ist —
     Schuljahresfrist, kein Ausleihlimit, unsichtbar im öffentlichen Katalog — war eine
     Konvention über Freitext, die zweimal in Produktion falsch lief. Jetzt ist es eine
     Entscheidung, die man sieht. -->
<script>
	import { klassenStufen } from '$lib/components/admin/buch_form_optionen.js';
	import Select from '../../../../lib/components/ui/Select.svelte';
	import Feld from '../../../../lib/components/ui/Feld.svelte';
	import Switch from '../../../../lib/components/ui/Switch.svelte';

	let { formular = $bindable(), systematikListe = [] } = $props();

	const faecher = $derived([
		{ value: '', label: 'Kein Fach' },
		...systematikListe.map((/** @type {any} */ s) => ({
			value: s.bezeichnung,
			label: `${s.kuerzel} - ${s.bezeichnung}`
		}))
	]);
	const klassen = klassenStufen.map((/** @type {number} */ k) => ({ value: k, label: String(k) }));
</script>

<div
	class="flex items-center justify-between gap-4 rounded-xl border border-outline-variant px-4 py-3"
>
	<div class="min-w-0">
		<label for="buch-lernmittel" class="block text-sm font-medium text-on-surface">Lernmittel</label
		>
		<p class="text-xs text-on-surface-variant">
			Schulbuch, das die Schule fürs Schuljahr leiht: Frist bis zum Stichtag, zählt nicht ins
			Ausleihlimit, erscheint nicht im öffentlichen Katalog.
		</p>
	</div>
	<Switch id="buch-lernmittel" bind:checked={formular.istLernmittel} />
</div>

<div class="grid grid-cols-2 gap-4">
	<div>
		<label for="buch-fach" class="mb-1.5 block text-sm font-medium text-on-surface-variant"
			>Fach</label
		>
		<Select
			id="buch-fach"
			bind:value={formular.subject}
			options={faecher}
			placeholder="Fach auswählen"
		/>
	</div>
	<div>
		<label for="buch-klasse" class="mb-1.5 block text-sm font-medium text-on-surface-variant"
			>Klasse</label
		>
		<Select id="buch-klasse" bind:value={formular.gradeLevel} options={klassen} />
	</div>
</div>

<div class="grid grid-cols-2 gap-4">
	<Feld
		id="buch-jahrgang-von"
		label="Geeignet für Jahrgang von"
		type="number"
		min="1"
		max="13"
		bind:value={formular.jahrgangVon}
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
