<script>
	import { onMount } from "svelte";
        import ClassAssignmentDialog from "./ClassAssignmentDialog.svelte";
	import KlassenKarte from "./KlassenKarte.svelte";
	import { csrfHeader } from "$lib/csrf.js";

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
			const res = await fetch(
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
			const res = await fetch(
				`/api/admin/class-books?className=${encodeURIComponent(className)}`,
				{
					method: "DELETE",
					credentials: "include",
					headers: /** @type {HeadersInit} */ ({
						...csrfHeader(),
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
		<h2 class="text-xl font-bold text-zinc-100 font-sans">Klassenübersicht</h2>
 
		<div class="flex gap-4 items-center">
			<select
				bind:value={filterBranch}
				onchange={loadGroups}
				class="bg-zinc-950 border border-zinc-800 text-zinc-300 py-2 px-3 rounded-lg shadow-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50 focus:border-emerald-500 text-sm cursor-pointer"
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
				class="bg-zinc-950 border border-zinc-800 text-zinc-300 py-2 px-3 rounded-lg shadow-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50 focus:border-emerald-500 text-sm cursor-pointer"
			>
				<option value="asc">Aufsteigend 5-10</option>
				<option value="desc">Absteigend 10-5</option>
			</select>
 
			<button
				onclick={() => {
					managingGroup = null;
					isManaging = true;
				}}
				class="px-4 py-2 bg-emerald-500 text-zinc-950 rounded-lg hover:bg-emerald-400 transition-colors text-sm font-bold flex items-center gap-2 shadow-lg shadow-emerald-955/20 cursor-pointer"
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
				class="animate-spin rounded-full h-10 w-10 border-b-2 border-emerald-500"
			></div>
		</div>
	{:else if error}
		<div class="text-red-400 border border-red-500/20 bg-red-500/10 text-center py-8 rounded-xl">
			{error}
		</div>
	{:else if classGroups.length === 0}
		<div
			class="text-center py-16 bg-zinc-955/40 rounded-2xl border border-dashed border-zinc-800/80"
		>
			<div class="text-4xl mb-4">📚</div>
			<h3 class="text-lg font-semibold text-zinc-200 mb-2">
				Noch keine Klassen angelegt
			</h3>
			<p class="text-zinc-500 text-sm max-w-md mx-auto">
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
