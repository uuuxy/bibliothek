<!-- @component KlassenUebersicht — der eine Ort für Klassensätze.

     Wer nur lesen darf, sieht diese Seite ohne die Aktionen. Deshalb liest sie über
     /api/class-books (view_books) statt über /api/admin/class-books (edit_books): Sonst
     liefe die Seite für die Helfer ins 403. -->
<script>
	import { BookOpen, Plus } from '@lucide/svelte';
	import Ladekreis from '../../../../lib/components/ui/Ladekreis.svelte';
	import LadeFehler from '../../../../lib/components/ui/LadeFehler.svelte';
	import SuchZustand from '../../../../lib/components/ui/SuchZustand.svelte';
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { loeschenBestaetigen } from '../../../../lib/stores/bestaetigung.svelte.js';
	import { showToast } from '$lib/store.svelte.js';
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

	// Höchstens eine Klasse ist ausgeklappt: Mit mehreren offenen wäre die Liste nach zwei
	// Klicks nicht mehr zu überblicken.
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
		// Ein neuer Versuch beginnt ohne den Fehler des letzten, sonst bliebe die Meldung stehen.
		error = null;
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
		if (!(await loeschenBestaetigen(`Buchliste von ${className} löschen?`))) return;
		try {
			const res = await apiFetch(
				`/api/admin/class-books?className=${encodeURIComponent(className)}`,
				{ method: 'DELETE', credentials: 'include' }
			);
			if (!res.ok) throw new Error('Fehler beim Löschen');
			loadGroups();
		} catch (err) {
			showToast(/** @type {any} */ (err).message, 'error');
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
		<div class="flex justify-center py-12"><Ladekreis size="lg" /></div>
	{:else if error}
		<LadeFehler
			titel="Klassensätze nicht geladen"
			text="Die Klassensätze konnten nicht abgerufen werden."
			onerneut={loadGroups}
		/>
	{:else if sichtbareGruppen.length === 0}
		<!-- Zwei verschiedene Leermeldungen: „nichts angelegt" schickt sonst jemanden
		     auf die Suche nach Daten, die es gibt — sein Suchbegriff passt nur nicht. -->
		{#if classGroups.length > 0}
			<SuchZustand
				symbol={BookOpen}
				titel="Keine Klasse gefunden"
				hinweis={`Zu „${klasseSearchQuery}" gibt es keinen Klassensatz. Suchfeld leeren zeigt wieder alle.`}
			/>
		{:else}
			<SuchZustand
				symbol={BookOpen}
				titel="Noch keine Klassen angelegt"
				hinweis={darfPflegen
					? 'Weise Bücher zu Klassen zu, um hier eine Übersicht zu sehen.'
					: 'Sobald Bücher einer Klasse zugewiesen sind, erscheinen die Klassensätze hier.'}
			/>
		{/if}
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
