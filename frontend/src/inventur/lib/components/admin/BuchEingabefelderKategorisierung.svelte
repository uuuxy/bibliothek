<script>
	import { klassenStufen, schulZweige } from '$lib/components/admin/buch_form_optionen.js';
	import Select from '../../../../lib/components/ui/Select.svelte';
	import Feld from '../../../../lib/components/ui/Feld.svelte';

	let { formular = $bindable(), systematikListe = [] } = $props();

	const nurBibliothek = $derived(formular.track === 'Bibliothek');

	const faecher = $derived([
		{ value: '', label: 'Kein Fach' },
		...systematikListe.map((/** @type {any} */ s) => ({
			value: s.bezeichnung,
			label: `${s.kuerzel} - ${s.bezeichnung}`
		}))
	]);
	const klassen = klassenStufen.map((/** @type {number} */ k) => ({ value: k, label: String(k) }));
	const zweige = schulZweige.map((/** @type {string} */ z) => ({ value: z, label: z }));
</script>

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
		<Select
			id="buch-klasse"
			bind:value={formular.gradeLevel}
			options={klassen}
			disabled={nurBibliothek}
		/>
	</div>
</div>

<div class="grid grid-cols-2 gap-4">
	<Feld
		id="buch-jahrgang-von"
		label="Verwendbar von Klasse"
		type="number"
		min="1"
		max="13"
		bind:value={formular.jahrgangVon}
		disabled={nurBibliothek}
	/>
	<Feld
		id="buch-jahrgang-bis"
		label="bis Klasse"
		type="number"
		min="1"
		max="13"
		bind:value={formular.jahrgangBis}
		disabled={nurBibliothek}
	/>
</div>

<div>
	<label for="buch-schulzweig" class="mb-1.5 block text-sm font-medium text-on-surface-variant"
		>Schulzweig</label
	>
	<Select id="buch-schulzweig" bind:value={formular.track} options={zweige} />
</div>
