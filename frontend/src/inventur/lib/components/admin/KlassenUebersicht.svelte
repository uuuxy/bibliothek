<script>
	import { BookOpen } from '@lucide/svelte';
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { onMount } from 'svelte';
	import ClassAssignmentDialog from './ClassAssignmentDialog.svelte';
	import KlassenKarte from './KlassenKarte.svelte';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import Select from '../../../../lib/components/ui/Select.svelte';

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

	async function loadGroups() {
		loading = true;
		try {
			const query = new URLSearchParams({
				branch: filterBranch,
				sort: sortOrder
			});
			const res = await apiFetch(`/api/admin/class-books?${query.toString()}`, {
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

<div class="space-y-10 py-6">
	<div class="flex justify-between items-center px-2">
		<h2 class="text-xl font-bold text-slate-800 font-sans">Klassenübersicht</h2>

		<div class="flex gap-4 items-center">
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

			<Button
				onclick={() => {
					managingGroup = null;
					isManaging = true;
				}}
			>
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"
					><path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 4v16m8-8H4"
					/></svg
				>
				Klasse hinzufügen
			</Button>
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
	{:else if classGroups.length === 0}
		<div class="text-center py-16 bg-slate-50/50 rounded-2xl border border-dashed border-slate-200">
			<div class="text-4xl mb-4"><BookOpen class="h-5 w-5" aria-hidden="true" /></div>
			<h3 class="text-lg font-semibold text-slate-800 mb-2">Noch keine Klassen angelegt</h3>
			<p class="text-slate-400 text-sm max-w-md mx-auto">
				Weise Bücher zu Klassen zu, um hier eine Übersicht zu sehen.
			</p>
		</div>
	{:else}
		{#each classGroups as group (group.className)}
			<KlassenKarte
				{group}
				onEdit={() => {
					managingGroup = group;
					isManaging = true;
				}}
				onDelete={() => deleteGroup(group.className)}
			/>
		{/each}
	{/if}
</div>

{#if isManaging}
	<ClassAssignmentDialog
		isOpen={isManaging}
		initialGroup={managingGroup}
		onClose={() => (isManaging = false)}
		onSaved={() => {
			isManaging = false;
			loadGroups();
		}}
	/>
{/if}
