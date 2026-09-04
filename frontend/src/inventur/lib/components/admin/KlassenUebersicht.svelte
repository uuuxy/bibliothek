<!-- @component KlassenUebersicht — der EINE Ort für Klassensätze.

     Bis zum 08.08.2026 gab es dieselbe Liste zweimal: hier mit Verwaltungsknöpfen
     und noch einmal als Reiter „Schulklassen"/„Klassensätze" im Medienkatalog, nur
     zum Blättern. Gleiche Daten (beide Routen landen im selben handleClassBooks),
     gleiche Darstellung, zwei gepflegte Komponentenpaare — und zwei Namen für eine
     Sache. Der Reiter ist weg; wer nur lesen darf, sieht diese Seite ohne die
     Aktionen. Deshalb liest sie über /api/class-books (view_books) statt über
     /api/admin/class-books (edit_books): Sonst liefe die Seite für genau die
     Helfer ins 403, denen der Reiter vorher offenstand. -->
<script>
	import { BookOpen, Plus } from '@lucide/svelte';
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { onMount } from 'svelte';
	import { authStore } from '../../../../lib/stores/authStore.svelte.js';
	import { hatRecht } from '../../../../lib/menu.js';
	import ClassAssignmentDialog from './ClassAssignmentDialog.svelte';
	import KlassenKarte from './KlassenKarte.svelte';
	import KlassenSuchfeld from '../KlassenSuchfeld.svelte';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import Select from '../../../../lib/components/ui/Select.svelte';

	const darfPflegen = $derived(hatRecht(authStore.currentUser, 'edit_books'));

	const ZWEIGE = [
		{ value: '', label: 'Alle anzeigen' },
		...['G', 'R', 'H', 'F'].map((z) => ({ value: z, label: `Nur ${z}-Klassen` }))
	];
	const SORTIERUNG = [
		{ value: 'asc', label: 'Aufsteigend 5-10' },
		{ value: 'desc', label: 'Absteigend 10-5' }
	];

	/** @type {any[]} */
	let classGroups = $state([]);
	let loading = $state(true);
	let error = $state(null);
	let isManaging = $state(false);
	let managingGroup = $state(null);

	let filterBranch = $state('');
	let sortOrder = $state('asc');
	let klasseSearchQuery = $state('');
	let isKlasseDropdownOpen = $state(false);

	// Der Name filtert direkt, statt einen Eintrag „auszuwählen": „5" zeigt alle
	// fünften Klassen, „5f1" genau eine. Die Vorschlagsliste setzt nur das Feld.
	// Set: Doppelte Namen wären doppelte each-Keys und rissen die Liste ab.
	const suchbegriff = $derived(klasseSearchQuery.trim().toLowerCase());
	const namen = $derived([...new Set(classGroups.map((g) => g.className.replace('Klasse ', '')))]);
	const filteredKlassenList = $derived(namen.filter((n) => n.toLowerCase().includes(suchbegriff)));
	const sichtbareGruppen = $derived(
		classGroups.filter((g) => g.className.toLowerCase().includes(suchbegriff))
	);

	// Hoechstens EINE Klasse ausgeklappt. Waeren mehrere offen, waere die Liste nach zwei
	// Klicks wieder so lang wie vorher — und genau das war Peters Einwand.
	let offeneKlasse = $state(/** @type {string|null} */ (null));

	// Bleibt nach dem Filtern genau eine Klasse uebrig, ist die Frage schon beantwortet:
	// Wer "5f1" eintippt, will diesen Satz sehen und nicht noch einmal klicken. Bei
	// mehreren Treffern bleibt alles zu, sonst waere die Uebersicht wieder verdeckt.
	$effect(() => {
		if (sichtbareGruppen.length === 1) {
			offeneKlasse = sichtbareGruppen[0].className;
		}
	});

	async function loadGroups() {
		loading = true;
		try {
			const query = new URLSearchParams({
				branch: filterBranch,
				sort: sortOrder
			});
			const res = await apiFetch(`/api/class-books?${query.toString()}`, {
				credentials: 'include'
			});
			if (!res.ok) throw new Error('Fehler beim Laden der Klassen-Bücher');
			const json = await res.json();
			classGroups = json.data || [];
		} catch (err) {
			error = /** @type {any} */ (err).message;
		} finally {
			loading = false;
		}
	}

	onMount(loadGroups);

	/**
	 * @param {string} className
	 */
	async function deleteGroup(className) {
		if (!confirm(`Klasse ${className} wirklich löschen?`)) return;
		try {
			const res = await apiFetch(
				`/api/admin/class-books?className=${encodeURIComponent(className)}`,
				{
					method: 'DELETE',
					credentials: 'include',
					headers: /** @type {HeadersInit} */ ({})
				}
			);
			if (!res.ok) throw new Error('Fehler beim Löschen');
			loadGroups();
		} catch (err) {
			alert(/** @type {any} */ (err).message);
		}
	}
</script>

<div class="space-y-10">
	<div class="flex flex-col gap-3">
		<KlassenSuchfeld
			bind:klasseSearchQuery
			bind:isKlasseDropdownOpen
			{filteredKlassenList}
			onSelectKlasse={(klasse) => {
				klasseSearchQuery = klasse;
				isKlasseDropdownOpen = false;
			}}
			class="w-full"
		/>

		<div class="flex flex-wrap gap-4 items-center">
			<Select
				bind:value={filterBranch}
				options={ZWEIGE}
				onchange={() => loadGroups()}
				class="w-44"
				aria-label="Klassen nach Zweig filtern"
			/>

			<Select
				bind:value={sortOrder}
				options={SORTIERUNG}
				onchange={() => loadGroups()}
				class="w-48"
				aria-label="Sortierung"
			/>

			{#if darfPflegen}
				<Button
					onclick={() => {
						managingGroup = null;
						isManaging = true;
					}}
				>
					<Plus class="w-4 h-4" aria-hidden="true" />
					Klasse hinzufügen
				</Button>
			{/if}
		</div>
	</div>

	{#if loading}
		<div class="flex justify-center py-12">
			<div class="animate-spin rounded-full h-10 w-10 border-b-2 border-blue-600"></div>
		</div>
	{:else if error}
		<div class="text-red-650 border border-red-200 bg-red-50 text-center py-8 rounded-xl">
			{error}
		</div>
	{:else if sichtbareGruppen.length === 0}
		<!-- Zwei verschiedene Leermeldungen: „nichts angelegt" schickt sonst jemanden
		     auf die Suche nach Daten, die es gibt — sein Suchbegriff passt nur nicht. -->
		<div class="text-center py-16 bg-slate-50/50 rounded-2xl border border-dashed border-slate-200">
			<div class="text-4xl mb-4"><BookOpen class="h-5 w-5" aria-hidden="true" /></div>
			{#if classGroups.length > 0}
				<h3 class="text-lg font-semibold text-slate-800 mb-2">Keine Klasse gefunden</h3>
				<p class="text-slate-400 text-sm max-w-md mx-auto">
					Zu „{klasseSearchQuery}" gibt es keinen Klassensatz. Suchfeld leeren zeigt wieder alle.
				</p>
			{:else}
				<h3 class="text-lg font-semibold text-slate-800 mb-2">Noch keine Klassen angelegt</h3>
				<p class="text-slate-400 text-sm max-w-md mx-auto">
					{darfPflegen
						? 'Weise Bücher zu Klassen zu, um hier eine Übersicht zu sehen.'
						: 'Sobald Bücher einer Klasse zugewiesen sind, erscheinen die Klassensätze hier.'}
				</p>
			{/if}
		</div>
	{:else}
		<div>
			{#each sichtbareGruppen as group (group.className)}
				<KlassenKarte
					{group}
					{darfPflegen}
					offen={offeneKlasse === group.className}
					onToggle={() =>
						(offeneKlasse = offeneKlasse === group.className ? null : group.className)}
					onEdit={() => {
						managingGroup = group;
						isManaging = true;
					}}
					onDelete={() => deleteGroup(group.className)}
				/>
			{/each}
		</div>
	{/if}
</div>

{#if isManaging}
	<ClassAssignmentDialog
		isOpen={isManaging}
		initialGroup={managingGroup}
		vorhandeneGruppen={classGroups}
		onClose={() => (isManaging = false)}
		onSaved={() => {
			isManaging = false;
			loadGroups();
		}}
	/>
{/if}
