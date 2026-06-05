<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { onMount } from "svelte";
        import ClassAssignmentDialog from "./ClassAssignmentDialog.svelte";
	import KlassenKarte from "./KlassenKarte.svelte";

	/** @type {any[]} */
	let classGroups = $state([]);
	let loading = $state(true);
	let error = $state(null);
	let isManaging = $state(false);
	let managingGroup = $state(null);

	let filterBranch = $state("");
	let sortOrder = $state("asc");

	async function loadGroups() {
		loading = true;
		try {
			const query = new URLSearchParams({
				branch: filterBranch,
				sort: sortOrder,
			});
			const res = await apiFetch(
				`/api/admin/class-books?${query.toString()}`,
				{
					credentials: "include",
				},
			);
			if (!res.ok)
				throw new Error("Fehler beim Laden der Klassen-Bücher");
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
					method: "DELETE",
					credentials: "include",
					headers: /** @type {HeadersInit} */ ({
					}),
				},
			);
			if (!res.ok) throw new Error("Fehler beim Löschen");
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
			<select
				bind:value={filterBranch}
				onchange={loadGroups}
				class="bg-white border border-slate-300 text-slate-700 py-2 px-3 rounded-lg shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-sm cursor-pointer"
			>
				<option value="">Alle anzeigen</option>
				<option value="G">Nur G-Klassen</option>
				<option value="R">Nur R-Klassen</option>
				<option value="H">Nur H-Klassen</option>
				<option value="F">Nur F-Klassen</option>
			</select>
 
			<select
				bind:value={sortOrder}
				onchange={loadGroups}
				class="bg-white border border-slate-300 text-slate-700 py-2 px-3 rounded-lg shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-sm cursor-pointer"
			>
				<option value="asc">Aufsteigend 5-10</option>
				<option value="desc">Absteigend 10-5</option>
			</select>
 
			<button
				onclick={() => {
					managingGroup = null;
					isManaging = true;
				}}
				class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-semibold flex items-center gap-2 shadow-sm cursor-pointer"
			>
				<svg
					class="w-4 h-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					><path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 4v16m8-8H4"
					/></svg
				>
				Klasse hinzufügen
			</button>
		</div>
	</div>
 
	{#if loading}
		<div class="flex justify-center py-12">
			<div
				class="animate-spin rounded-full h-10 w-10 border-b-2 border-blue-600"
			></div>
		</div>
	{:else if error}
		<div class="text-red-650 border border-red-200 bg-red-50 text-center py-8 rounded-xl">
			{error}
		</div>
	{:else if classGroups.length === 0}
		<div
			class="text-center py-16 bg-slate-50/50 rounded-2xl border border-dashed border-slate-200"
		>
			<div class="text-4xl mb-4">📚</div>
			<h3 class="text-lg font-semibold text-slate-800 mb-2">
				Noch keine Klassen angelegt
			</h3>
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
