<script>
	import { klassenStufen, schulZweige } from '$lib/components/admin/buch_form_optionen.js';
	import Select from '../../../../lib/components/ui/Select.svelte';

	let { formular = $bindable(), systematikListe = [] } = $props();

	const nurBibliothek = $derived(formular.track === 'Bibliothek');
	const gesperrt = $derived(nurBibliothek ? 'opacity-50 cursor-not-allowed' : '');

	const faecher = $derived([
		{ value: '', label: 'Kein Fach' },
		...systematikListe.map((/** @type {any} */ s) => ({
			value: s.bezeichnung,
			label: `${s.kuerzel} - ${s.bezeichnung}`
		}))
	]);
	const klassen = klassenStufen.map((/** @type {string} */ k) => ({ value: k, label: k }));
	const zweige = schulZweige.map((/** @type {string} */ z) => ({ value: z, label: z }));
</script>

<div class="grid grid-cols-2 gap-4">
	<div>
		<label for="buch-fach" class="block text-sm font-medium text-slate-700 mb-1">Fach</label>
		<Select
			id="buch-fach"
			bind:value={formular.subject}
			options={faecher}
			placeholder="Fach auswählen"
		/>
	</div>
	<div>
		<label for="buch-klasse" class="block text-sm font-medium text-slate-700 mb-1">Klasse</label>
		<Select
			id="buch-klasse"
			bind:value={formular.gradeLevel}
			options={klassen}
			disabled={nurBibliothek}
		/>
	</div>
</div>

<div class="grid grid-cols-2 gap-4">
	<div>
		<label for="buch-jahrgang-von" class="block text-sm font-medium text-slate-700 mb-1"
			>Verwendbar von Klasse</label
		>
		<input
			id="buch-jahrgang-von"
			type="number"
			min="1"
			max="13"
			bind:value={formular.jahrgangVon}
			disabled={nurBibliothek}
			class="w-full rounded-lg border-slate-300 bg-slate-50 px-4 py-2.5 text-slate-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition {gesperrt}"
		/>
	</div>
	<div>
		<label for="buch-jahrgang-bis" class="block text-sm font-medium text-slate-700 mb-1"
			>bis Klasse</label
		>
		<input
			id="buch-jahrgang-bis"
			type="number"
			min="1"
			max="13"
			bind:value={formular.jahrgangBis}
			disabled={nurBibliothek}
			class="w-full rounded-lg border-slate-300 bg-slate-50 px-4 py-2.5 text-slate-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition {gesperrt}"
		/>
	</div>
</div>

<div>
	<label for="buch-schulzweig" class="block text-sm font-medium text-slate-700 mb-1"
		>Schulzweig</label
	>
	<Select id="buch-schulzweig" bind:value={formular.track} options={zweige} />
</div>
