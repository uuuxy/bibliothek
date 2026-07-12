<!--
  @component
  DeletedStudentList

  Diese Komponente stellt die Ansicht für den Papierkorb (gelöschte Schüler) dar.
  Sie lädt die Daten asynchron vom Backend und bietet eine Wiederherstellungsfunktion.
-->
<script>
	import { apiFetch } from '../../apiFetch.js';
	import { onMount } from 'svelte';

	let { onRestoreSuccess = () => {} } = $props();

	/** @type {any[]} */
	let deletedStudents = $state.raw([]);
	let loadingDeleted = $state(false);

	export async function loadDeletedStudents() {
		loadingDeleted = true;
		try {
			const res = await apiFetch('/api/schueler/deleted');
			if (res.ok) {
				deletedStudents = (await res.json()) || [];
			}
		} catch (err) {
			console.error('Fehler beim Laden des Papierkorbs:', err);
		} finally {
			loadingDeleted = false;
		}
	}

	async function restoreStudent(/** @type {string} */ id) {
		try {
			const res = await apiFetch(`/api/schueler/${id}/restore`, { method: 'POST' });
			if (res.ok) {
				loadDeletedStudents();
				onRestoreSuccess();
			}
		} catch (err) {
			console.error('Fehler bei Wiederherstellung:', err);
		}
	}

	onMount(() => {
		loadDeletedStudents();
	});
</script>

<div class="w-full border-l-2 border-rose-300">
	<div class="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
		<h3 class="text-base font-bold text-rose-800 flex items-center gap-2">
			<svg
				xmlns="http://www.w3.org/2000/svg"
				class="h-5 w-5"
				viewBox="0 0 20 20"
				fill="currentColor"
				><path
					fill-rule="evenodd"
					d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z"
					clip-rule="evenodd"
				/></svg
			>
			Gelöschte Schüler (Papierkorb)
		</h3>
	</div>

	{#if loadingDeleted}
		<div class="py-16 flex justify-center items-center">
			<div
				class="w-8 h-8 border-4 border-t-rose-600 border-slate-200 rounded-full animate-spin"
				aria-hidden="true"
			></div>
		</div>
	{:else if deletedStudents.length === 0}
		<div class="py-16 flex flex-col items-center justify-center text-slate-400 space-y-2">
			<svg
				xmlns="http://www.w3.org/2000/svg"
				class="h-10 w-10 text-slate-300"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				aria-hidden="true"
				><path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="1.5"
					d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
				/></svg
			>
			<span class="text-xs font-semibold">Der Papierkorb ist leer.</span>
		</div>
	{:else}
		<div class="overflow-x-auto w-full text-left">
			<table class="w-full text-base text-slate-700">
				<thead class="border-b border-gray-200 text-sm font-semibold text-gray-500 font-sans">
					<tr>
						<th class="px-6 py-4">Name</th>
						<th class="px-6 py-4 w-24">Klasse</th>
						<th class="px-6 py-4 w-44">Gelöscht am</th>
						<th class="px-6 py-4 w-36 text-right">Aktion</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each deletedStudents as s, _i (_i)}
						<tr class="hover:bg-slate-50/50 transition-colors">
							<td class="px-6 py-3 font-semibold text-slate-800">
								{s.vorname}
								{s.nachname}
								<div class="text-[9px] text-slate-400 font-normal mt-0.5">{s.barcode_id}</div>
							</td>
							<td class="px-6 py-3 font-medium text-slate-600">
								Kl. {s.klasse || 'N/A'}
							</td>
							<td class="px-6 py-3 text-sm text-slate-500">
								{new Date(s.deleted_at).toLocaleString('de-DE', {
									day: '2-digit',
									month: '2-digit',
									year: 'numeric',
									hour: '2-digit',
									minute: '2-digit'
								})}
							</td>
							<td class="px-6 py-3 text-right">
								<button
									onclick={() => restoreStudent(s.id)}
									title="Wiederherstellen"
									class="inline-flex items-center justify-center w-8 h-8 rounded-lg bg-emerald-100 text-emerald-700 hover:bg-emerald-200 transition-colors shadow-sm cursor-pointer"
								>
									<svg
										xmlns="http://www.w3.org/2000/svg"
										class="h-4.5 w-4.5"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="2.5"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6"
										/>
									</svg>
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>
