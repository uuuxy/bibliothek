<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';

	/** Öffnet das Profil des überfälligen Schülers in der Schülerdatei (zentraler Request). */
	function openProfile(schuelerId) {
		uiStore.requestedStudentId = schuelerId;
		uiStore.activeTab = 'students_dir';
	}

	// Derived state for 'Select All' checkbox
	let allSelected = $derived(
		mahnwesenStore.filteredSchueler.length > 0 &&
			mahnwesenStore.selectedIds.size === mahnwesenStore.filteredSchueler.length
	);

	let indeterminate = $derived(
		mahnwesenStore.selectedIds.size > 0 &&
			mahnwesenStore.selectedIds.size < mahnwesenStore.filteredSchueler.length
	);

	// Toggle all
	function toggleAll() {
		if (allSelected) mahnwesenStore.deselectAllSchueler();
		else mahnwesenStore.selectAllSchueler();
	}
</script>

{#if mahnwesenStore.loading}
	<div class="flex justify-center py-20">
		<div
			class="w-8 h-8 border-4 border-blue-500/30 border-t-blue-500 rounded-full animate-spin"
		></div>
	</div>
{:else if mahnwesenStore.error}
	<div
		class="bg-rose-50 border border-rose-200 rounded-2xl p-6 text-center text-rose-600 text-sm font-medium"
	>
		{mahnwesenStore.error}
	</div>
{:else if !mahnwesenStore.data || mahnwesenStore.klassen.length === 0}
	<div class="bg-emerald-50 border border-emerald-200 rounded-2xl p-10 text-center">
		<p class="text-emerald-700 font-semibold">Keine überfälligen Ausleihen vorhanden. 🎉</p>
	</div>
{:else}
	<!-- MD3 Table -->
	<div class="bg-white w-full pb-6">
		<div class="overflow-x-auto w-full">
			<table class="w-full text-left text-sm whitespace-nowrap">
				<thead class="bg-slate-50 border-b border-slate-200 text-slate-500 font-medium">
					<tr>
						<th class="w-12 px-4 py-3 text-center">
							<input
								type="checkbox"
								class="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500/20 transition-all cursor-pointer"
								checked={allSelected}
								{indeterminate}
								onclick={toggleAll}
							/>
						</th>
						<th class="px-4 py-3">Schüler/in</th>
						<th class="px-4 py-3">Klasse</th>
						<th class="px-4 py-3">Medien</th>
						<th class="px-4 py-3">Status</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each mahnwesenStore.filteredSchueler as schueler, _i (_i)}
						<tr
							class="hover:bg-slate-50 transition-colors {mahnwesenStore.selectedIds.has(
								schueler.schueler_id
							)
								? 'bg-blue-50/50'
								: ''}"
						>
							<td class="w-12 px-4 py-3 text-center">
								<input
									type="checkbox"
									class="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500/20 transition-all cursor-pointer"
									checked={mahnwesenStore.selectedIds.has(schueler.schueler_id)}
									onclick={() => mahnwesenStore.toggleSelect(schueler.schueler_id)}
								/>
							</td>
							<td class="px-4 py-3">
								<div class="flex items-center gap-1.5">
									<button
										type="button"
										onclick={() => openProfile(schueler.schueler_id)}
										class="font-semibold text-slate-800 text-left hover:text-blue-700 hover:underline cursor-pointer rounded focus-visible:outline-2 focus-visible:outline-blue-600"
										aria-label="Profil von {schueler.name} anzeigen"
									>
										{schueler.name}
									</button>
									{#if !schueler.eltern_email}
										<!-- Dezentes „keine Eltern-E-Mail"-Icon statt lautem Dauer-Label auf jeder Zeile. -->
										<span
											class="text-slate-300 shrink-0 flex items-center"
											title="Keine Eltern-E-Mail hinterlegt"
											aria-label="Keine Eltern-E-Mail hinterlegt"
										>
											<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
												<path stroke-linecap="round" stroke-linejoin="round" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
												<path stroke-linecap="round" stroke-linejoin="round" d="M3 3l18 18" />
											</svg>
										</span>
									{/if}
								</div>
							</td>
							<td class="px-4 py-3">
								<span
									class="inline-flex items-center px-2 py-1 rounded-md bg-slate-100 text-slate-700 font-bold text-xs"
								>
									{schueler.klasse}
								</span>
							</td>
							<td class="px-4 py-3">
								<div class="flex -space-x-2">
									{#each schueler.medien.slice(0, 3) as medium, _i (_i)}
										{#if medium.cover_url}
											<img
												src={medium.cover_url}
												alt="Cover"
												class="w-8 h-10 rounded-md border-2 border-white object-cover shadow-sm"
												loading="lazy"
												title={medium.titel}
											/>
										{:else}
											<div
												class="w-8 h-10 rounded-md border-2 border-white bg-slate-200 shadow-sm flex items-center justify-center text-[8px] text-slate-500 font-bold"
												title={medium.titel}
											>
												?
											</div>
										{/if}
									{/each}
									{#if schueler.medien.length > 3}
										<div
											class="w-8 h-10 rounded-md border-2 border-white bg-slate-100 flex items-center justify-center text-[10px] font-bold text-slate-600 shadow-sm z-10"
										>
											+{schueler.medien.length - 3}
										</div>
									{/if}
								</div>
							</td>
							<td class="px-4 py-3">
								<div class="flex flex-col items-start gap-1">
									<span
										class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-bold
                  {schueler.mahnstufe === 'Mahnung'
											? 'bg-rose-100 text-rose-700'
											: schueler.mahnstufe === '1. Erinnerung'
												? 'bg-amber-100 text-amber-800'
												: schueler.mahnstufe === 'Lehrerkollegium'
													? 'bg-blue-100 text-blue-700'
													: 'bg-emerald-50 text-emerald-700'}"
									>
										{schueler.mahnstufe}
									</span>
									<span
										class="text-[11px] font-medium {schueler.maxTage > 14
											? 'text-rose-500'
											: 'text-slate-400'}"
									>
										{schueler.maxTage === 0
											? 'heute fällig'
											: `${schueler.maxTage} ${schueler.maxTage === 1 ? 'Tag' : 'Tage'} überfällig`}
									</span>
								</div>
							</td>
						</tr>
					{/each}
					{#if mahnwesenStore.filteredSchueler.length === 0}
						<tr>
							<td colspan="5" class="px-4 py-8 text-center text-slate-500">
								Keine Treffer für die aktuelle Auswahl (Tab, Klasse oder Suche).
							</td>
						</tr>
					{/if}
				</tbody>
			</table>
		</div>
	</div>
{/if}

<!-- Die Auswahl-Aktion („Mahnbriefe drucken") lebt jetzt in der kontextuellen Toolbar oben
     (MahnwesenFilters) — kein separater Schwebe-Balken mehr. -->

<!-- E-Mail Modal -->
{#if mahnwesenStore.modalOpen}
	<div class="fixed inset-0 z-60 flex items-center justify-center p-4">
		<div
			class="absolute inset-0 bg-black/20 backdrop-blur-sm"
			onclick={mahnwesenStore.closeModal}
			aria-hidden="true"
		></div>
		<div
			class="relative bg-white rounded-2xl shadow-2xl border border-slate-200 w-full max-w-md p-6 space-y-5"
		>
			<div class="flex items-center justify-between">
				<h2 class="text-base font-bold text-slate-800">Mahnliste per E-Mail senden</h2>
				<button
					onclick={mahnwesenStore.closeModal}
					aria-label="Modal schließen"
					class="p-1.5 rounded-lg text-slate-400 hover:bg-slate-100 transition-colors"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-4 w-4"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2.5"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<div class="space-y-4">
				<div>
					<span class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1"
						>Klasse</span
					>
					<p class="text-sm font-semibold text-slate-800">{mahnwesenStore.modalKlasse}</p>
				</div>
				<div>
					<label
						for="modal-email"
						class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1"
						>E-Mail-Adresse des Klassenlehrers</label
					>
					<input
						id="modal-email"
						type="email"
						bind:value={mahnwesenStore.modalEmail}
						placeholder="lehrer@schule.de"
						class="w-full px-3 py-2.5 rounded-xl border border-slate-200 bg-slate-50 text-sm text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all"
					/>
					{#if !mahnwesenStore.modalEmail.trim()}
						<p class="text-[10px] text-slate-400 mt-1">
							Die Adresse wird aus dem Klassenlehrer-Mapping vorausgefüllt, kann aber geändert
							werden.
						</p>
					{/if}
				</div>
			</div>

			{#if mahnwesenStore.modalMsg}
				<div
					class="rounded-xl px-4 py-3 text-xs font-semibold {mahnwesenStore.modalMsg.type ===
					'success'
						? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
						: 'bg-rose-50 text-rose-600 border border-rose-200'}"
				>
					{mahnwesenStore.modalMsg.text}
				</div>
			{/if}

			<div class="flex justify-end gap-2">
				<button
					onclick={mahnwesenStore.closeModal}
					class="px-4 py-2 rounded-xl border border-slate-200 text-slate-600 hover:bg-slate-50 text-xs font-semibold transition-all"
					>Abbrechen</button
				>
				<button
					onclick={mahnwesenStore.sendMahnliste}
					disabled={mahnwesenStore.modalSending || mahnwesenStore.modalMsg?.type === 'success'}
					class="px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-xs font-bold transition-all flex items-center gap-2"
				>
					{#if mahnwesenStore.modalSending}
						<div
							class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"
						></div>
					{:else}
						<svg
							xmlns="http://www.w3.org/2000/svg"
							class="h-3.5 w-3.5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2.5"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
							/>
						</svg>
					{/if}
					Senden
				</button>
			</div>
		</div>
	</div>
{/if}
