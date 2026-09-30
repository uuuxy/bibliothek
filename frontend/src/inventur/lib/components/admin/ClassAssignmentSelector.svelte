<script>
	// Die Zielklassen des Klassensatz-Dialogs als Chips. Auswählen statt tippen (docs/OFFEN.md
	// 5.18, 30.09.2026): Bis dahin tippte man die Klassen frei ein, und ein Tippfehler legte
	// eine Klasse an, die es an der Schule nicht gibt. Zur Wahl stehen die Klassen der Schüler
	// und die Klassen, die schon eine Buchliste haben (wie in ClassAssignPicker).
	import { onMount } from 'svelte';
	import { X } from '@lucide/svelte';
	import Select from '../../../../lib/components/ui/Select.svelte';
	import { erzeugeKlassenVorschlaege } from '../../../../lib/components/students/klassenVorschlaege.svelte.js';

	/** @type {{ selectedClasses: string[], vorhandeneGruppen?: { className: string }[] }} */
	let { selectedClasses = $bindable([]), vorhandeneGruppen = [] } = $props();

	const klassenListe = erzeugeKlassenVorschlaege();
	onMount(klassenListe.lade);

	const optionen = $derived(
		[...new Set([...klassenListe.liste, ...vorhandeneGruppen.map((g) => g.className)])]
			.filter((k) => !selectedClasses.includes(k))
			.sort()
			.map((k) => ({ value: k, label: k }))
	);

	/** Das Auswahlfeld fügt hinzu und steht danach wieder leer. */
	let auswahl = $state('');

	/** @param {string} name */
	function hinzufuegen(name) {
		if (name && !selectedClasses.includes(name)) selectedClasses = [...selectedClasses, name];
		auswahl = '';
	}

	/** @param {string} name */
	function removeClass(name) {
		selectedClasses = selectedClasses.filter((c) => c !== name);
	}
</script>

<div class="mb-4 grid gap-y-1.5 sm:mb-6">
	<label for="class-input" class="text-sm font-medium text-on-surface-variant">Zielklassen</label>
	<div class="flex flex-wrap items-center gap-2">
		{#each selectedClasses as selectedClass (selectedClass)}
			<span
				class="inline-flex items-center gap-1.5 px-4 py-1.5 bg-secondary-container text-on-secondary-container rounded-full text-sm font-semibold"
			>
				{selectedClass}
				<button
					onclick={() => removeClass(selectedClass)}
					class="hover:opacity-70 rounded-full transition-opacity ml-1"
					aria-label="Klasse {selectedClass} entfernen"
					title="Klasse entfernen"
				>
					<X class="w-4 h-4" aria-hidden="true" />
				</button>
			</span>
		{/each}
		<Select
			id="class-input"
			bind:value={auswahl}
			options={optionen}
			placeholder={optionen.length ? 'Klasse wählen' : 'Keine weiteren Klassen'}
			onchange={hinzufuegen}
			class="w-48"
		/>
	</div>
</div>
