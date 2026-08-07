<!--
  @component
  DeletedStudentList

  Diese Komponente stellt die Ansicht für den Papierkorb (gelöschte Schüler) dar.
  Sie lädt die Daten asynchron vom Backend und bietet eine Wiederherstellungsfunktion.
-->
<script>
	import { apiFetch } from '../../apiFetch.js';
	import { onMount } from 'svelte';
	import Sheet from '../layout/Sheet.svelte';
	import { Trash2, Undo2 } from '@lucide/svelte';

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

<Sheet class="border-l-4 border-l-rose-400">
	<div class="px-6 py-4 border-b border-slate-200 flex items-center justify-between">
		<h3 class="text-base font-bold text-rose-800 flex items-center gap-2">
			<Trash2 class="h-5 w-5" aria-hidden="true" />
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
			<Trash2 class="h-10 w-10 text-slate-300" aria-hidden="true" />
			<span class="text-xs font-semibold">Der Papierkorb ist leer.</span>
		</div>
	{:else}
		<div class="overflow-x-auto w-full text-left">
			<table class="w-full text-base text-slate-700">
				<thead class="border-b border-slate-200 text-sm font-semibold text-slate-500 font-sans">
					<tr>
						<th class="px-4 py-2">Name</th>
						<th class="px-4 py-2 w-24">Klasse</th>
						<th class="px-4 py-2 w-44">Gelöscht am</th>
						<th class="px-4 py-2 w-36 text-right">Aktion</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each deletedStudents as s, _i (_i)}
						<tr class="hover:bg-slate-50/50 transition-colors">
							<td class="px-4 py-2 font-semibold text-slate-800">
								{s.vorname}
								{s.nachname}
								<div class="text-label-small text-slate-400 font-normal mt-0.5">{s.barcode_id}</div>
							</td>
							<td class="px-4 py-2 font-medium text-slate-600">
								Kl. {s.klasse || 'N/A'}
							</td>
							<td class="px-4 py-2 text-sm text-slate-500">
								{new Date(s.deleted_at).toLocaleString('de-DE', {
									day: '2-digit',
									month: '2-digit',
									year: 'numeric',
									hour: '2-digit',
									minute: '2-digit'
								})}
							</td>
							<td class="px-4 py-2 text-right">
								<button
									onclick={() => restoreStudent(s.id)}
									title="Wiederherstellen"
									aria-label="Wiederherstellen"
									class="inline-flex items-center justify-center w-8 h-8 rounded-lg bg-emerald-100 text-emerald-700 hover:bg-emerald-200 transition-colors shadow-sm cursor-pointer"
								>
									<Undo2 class="h-4.5 w-4.5" aria-hidden="true" />
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</Sheet>
