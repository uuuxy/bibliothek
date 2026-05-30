<script>
	import { onMount } from 'svelte';

	let loading = $state(true);
	let saving = $state(false);
	let toast = $state(/** @type {{msg:string,type:string}|null} */ (null));

	let ferienLeseclubAktiv = $state(false);
	let ferienLeseclubZieldatum = $state('');
	let lmfStichtag = $state('07-31');

	// ── Klassenlehrer-Mapping ──────────────────────────────────────────
	let mappingRows = $state(/** @type {{klasse:string, lehrer_email:string, erstellt_am?:string}[]} */ ([]));
	let mappingLoading = $state(false);
	let newMappingKlasse = $state('');
	let newMappingEmail = $state('');
	let mappingSaving = $state(false);

	function showToast(msg, type = 'success') {
		toast = { msg, type };
		setTimeout(() => { toast = null; }, 3500);
	}

	async function fetchMapping() {
		mappingLoading = true;
		try {
			const res = await fetch('/api/klassen-mapping');
			if (res.ok) mappingRows = await res.json();
		} catch { /* ignore */ } finally {
			mappingLoading = false;
		}
	}

	async function upsertMapping() {
		if (!newMappingKlasse.trim() || !newMappingEmail.trim()) return;
		mappingSaving = true;
		try {
			const res = await fetch('/api/klassen-mapping', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ klasse: newMappingKlasse.trim(), lehrer_email: newMappingEmail.trim() })
			});
			if (res.ok) {
				newMappingKlasse = '';
				newMappingEmail = '';
				await fetchMapping();
				showToast('Mapping gespeichert.');
			} else {
				showToast(await res.text() || 'Fehler beim Speichern', 'error');
			}
		} catch {
			showToast('Netzwerkfehler', 'error');
		} finally {
			mappingSaving = false;
		}
	}

	async function deleteMapping(klasse) {
		try {
			const res = await fetch(`/api/klassen-mapping/${encodeURIComponent(klasse)}`, { method: 'DELETE' });
			if (res.ok || res.status === 204) {
				await fetchMapping();
				showToast(`Mapping für ${klasse} gelöscht.`);
			} else {
				showToast('Fehler beim Löschen', 'error');
			}
		} catch {
			showToast('Netzwerkfehler', 'error');
		}
	}

	onMount(async () => {
		try {
			const res = await fetch('/api/einstellungen');
			if (res.ok) {
				const data = await res.json();
				ferienLeseclubAktiv = data.ferien_leseclub_aktiv ?? false;
				ferienLeseclubZieldatum = data.ferien_leseclub_zieldatum ?? '';
				lmfStichtag = data.lmf_stichtag ?? '07-31';
			}
		} catch { /* use defaults */ }
		await fetchMapping();
		loading = false;
	});

	async function saveSettings() {
		saving = true;
		try {
			const res = await fetch('/api/einstellungen', {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					ferien_leseclub_aktiv: ferienLeseclubAktiv,
					ferien_leseclub_zieldatum: ferienLeseclubZieldatum || null,
					lmf_stichtag: lmfStichtag || '07-31'
				})
			});
			if (res.ok) {
				showToast('Einstellungen gespeichert.');
			} else {
				showToast(await res.text() || 'Fehler beim Speichern', 'error');
			}
		} catch {
			showToast('Netzwerkfehler', 'error');
		}
		saving = false;
	}
</script>

<div class="max-w-2xl mx-auto space-y-6">
	<h1 class="text-2xl font-bold text-gray-900">Einstellungen</h1>

	{#if loading}
		<div class="text-gray-400 text-sm py-8 text-center">Lade Einstellungen…</div>
	{:else}
		<!-- ── Ferien-Leseclub ── -->
		<div class="bg-white p-6 rounded-2xl shadow-sm border border-gray-200 space-y-5">
			<div class="flex items-center justify-between">
				<div>
					<h3 class="text-lg font-semibold text-gray-900">Ferien-Leseclub</h3>
					<p class="text-sm text-gray-500 mt-0.5">Wenn aktiv, wird bei jeder Ausleihe das Rückgabedatum pauschal auf das unten definierte Ferienende gesetzt – alle Standardfristen werden überschrieben.</p>
				</div>
				<!-- Toggle switch -->
				<button
					type="button"
					onclick={() => (ferienLeseclubAktiv = !ferienLeseclubAktiv)}
					class="relative inline-flex h-7 w-12 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 focus:outline-none {ferienLeseclubAktiv ? 'bg-emerald-500' : 'bg-gray-200'}"
					role="switch"
					aria-checked={ferienLeseclubAktiv}>
					<span class="pointer-events-none inline-block h-6 w-6 rounded-full bg-white shadow transition-transform duration-200 {ferienLeseclubAktiv ? 'translate-x-5' : 'translate-x-0'}"></span>
				</button>
			</div>

			{#if ferienLeseclubAktiv}
				<div class="p-4 rounded-xl bg-emerald-50 border border-emerald-100 space-y-3 animate-slide-up">
					<div class="flex items-center gap-2 text-emerald-700 text-sm font-semibold">
						<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
						Ferien-Leseclub ist aktiv
					</div>
					<label class="block">
						<span class="text-sm font-medium text-gray-700">Ferienende (Rückgabezieldatum)</span>
						<input
							type="date"
							bind:value={ferienLeseclubZieldatum}
							class="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-emerald-400 focus:ring-2 focus:ring-emerald-100 focus:outline-none" />
						<p class="text-xs text-gray-400 mt-1">Alle Ausleihen bekommen dieses Datum als Rückgabefrist.</p>
					</label>
				</div>
			{/if}
		</div>

		<!-- ── LMF-Stichtag ── -->
		<div class="bg-white p-6 rounded-2xl shadow-sm border border-gray-200 space-y-4">
			<h3 class="text-lg font-semibold text-gray-900">LMF-Stichtag</h3>
			<p class="text-sm text-gray-500">Bücher, deren Titel mit <code class="bg-gray-100 px-1 rounded">lmf-</code> beginnen, erhalten dieses Datum als Rückgabefrist (Schuljahresende).</p>
			<label class="block">
				<span class="text-sm font-medium text-gray-700">Stichtag (MM-TT)</span>
				<input
					type="text"
					bind:value={lmfStichtag}
					placeholder="07-31"
					pattern="\d{2}-\d{2}"
					maxlength="5"
					class="mt-1 block w-40 rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none" />
				<p class="text-xs text-gray-400 mt-1">Format: MM-TT (z. B. <code>07-31</code> für 31. Juli)</p>
			</label>
		</div>

		<!-- ── Fristen-Übersicht ── -->
		<div class="bg-white p-6 rounded-2xl shadow-sm border border-gray-200">
			<h3 class="text-lg font-semibold text-gray-900 mb-3">Standardfristen</h3>
			<div class="grid grid-cols-3 gap-3">
				<div class="rounded-xl bg-slate-50 border border-slate-100 p-4 text-center">
					<div class="text-2xl font-bold text-slate-700">21</div>
					<div class="text-xs text-slate-500 mt-1 font-medium">Tage · Buch</div>
				</div>
				<div class="rounded-xl bg-amber-50 border border-amber-100 p-4 text-center">
					<div class="text-2xl font-bold text-amber-600">7</div>
					<div class="text-xs text-amber-600 mt-1 font-medium">Tage · CD / DVD</div>
				</div>
				<div class="rounded-xl bg-blue-50 border border-blue-100 p-4 text-center">
					<div class="text-sm font-bold text-blue-600">{lmfStichtag || '07-31'}</div>
					<div class="text-xs text-blue-500 mt-1 font-medium">Datum · LMF</div>
				</div>
			</div>
		</div>

		<!-- ── Klassenlehrer-Mapping ── -->
		<div class="bg-white p-6 rounded-2xl shadow-sm border border-gray-200 space-y-4">
			<div>
				<h3 class="text-lg font-semibold text-gray-900">Klassenlehrer-Mapping</h3>
				<p class="text-sm text-gray-500 mt-0.5">Weist jeder Klasse die E-Mail-Adresse der Klassenlehrerin / des Klassenlehrers zu. Diese Adresse wird im Mahnwesen vorausgefüllt.</p>
			</div>

			{#if mappingLoading}
				<p class="text-sm text-gray-400">Lade Mapping…</p>
			{:else if mappingRows.length === 0}
				<p class="text-sm text-gray-400 italic">Noch keine Einträge vorhanden.</p>
			{:else}
				<div class="overflow-hidden rounded-xl border border-gray-200">
					<table class="w-full text-sm">
						<thead class="bg-gray-50 text-gray-500">
							<tr>
								<th class="text-left px-4 py-2.5 font-semibold text-xs uppercase tracking-wide">Klasse</th>
								<th class="text-left px-4 py-2.5 font-semibold text-xs uppercase tracking-wide">E-Mail Klassenlehrer</th>
								<th class="px-4 py-2.5 w-12"></th>
							</tr>
						</thead>
						<tbody class="divide-y divide-gray-100">
							{#each mappingRows as row (row.klasse)}
								<tr class="hover:bg-gray-50 transition-colors">
									<td class="px-4 py-2.5 font-mono font-semibold text-slate-700">{row.klasse}</td>
									<td class="px-4 py-2.5 text-slate-600">{row.lehrer_email}</td>
									<td class="px-4 py-2.5 text-right">
										<button
											onclick={() => deleteMapping(row.klasse)}
											class="p-1.5 rounded-lg text-rose-400 hover:bg-rose-50 hover:text-rose-600 transition-colors"
											title="Löschen"
										>
											<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
												<path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
											</svg>
										</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}

			<!-- Add new mapping -->
			<div class="flex gap-3 items-end pt-2 border-t border-gray-100">
				<div class="w-28">
					<label class="block text-[10px] font-bold text-gray-400 uppercase tracking-wider mb-1">Klasse</label>
					<input
						type="text"
						bind:value={newMappingKlasse}
						placeholder="z. B. 8b"
						class="w-full px-3 py-2 rounded-lg border border-gray-300 text-sm focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none font-mono"
					/>
				</div>
				<div class="flex-1">
					<label class="block text-[10px] font-bold text-gray-400 uppercase tracking-wider mb-1">E-Mail</label>
					<input
						type="email"
						bind:value={newMappingEmail}
						placeholder="klassenlehrer@schule.de"
						class="w-full px-3 py-2 rounded-lg border border-gray-300 text-sm focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none"
					/>
				</div>
				<button
					onclick={upsertMapping}
					disabled={mappingSaving || !newMappingKlasse.trim() || !newMappingEmail.trim()}
					class="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-semibold transition-colors flex items-center gap-1.5 shrink-0"
				>
					{#if mappingSaving}
						<div class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"></div>
					{:else}
						Speichern
					{/if}
				</button>
			</div>
		</div>

		<!-- ── System ── -->
		<div class="bg-white p-6 rounded-2xl shadow-sm border border-gray-200 space-y-4">
			<h3 class="text-lg font-semibold text-gray-900">System</h3>
			<p class="text-gray-500 text-sm">Version 1.0.0</p>
			<div class="pt-2 border-t border-gray-100">
				<button class="px-4 py-2 bg-red-50 text-red-600 rounded-lg text-sm font-medium hover:bg-red-100 transition-colors">
					Datenbank zurücksetzen
				</button>
			</div>
		</div>

		<div class="flex justify-end">
			<button
				onclick={saveSettings}
				disabled={saving}
				class="px-6 py-2.5 bg-blue-600 text-white font-semibold rounded-xl hover:bg-blue-700 disabled:opacity-50 transition-colors shadow-sm">
				{saving ? 'Wird gespeichert…' : 'Einstellungen speichern'}
			</button>
		</div>
	{/if}
</div>

{#if toast}
	<div class="fixed bottom-6 right-6 px-5 py-3 rounded-xl shadow-xl text-sm font-semibold z-50 animate-slide-up {toast.type === 'error' ? 'bg-rose-600 text-white' : 'bg-emerald-600 text-white'}">
		{toast.msg}
	</div>
{/if}

<style>
	@keyframes slide-up {
		from { opacity: 0; transform: translateY(8px); }
		to   { opacity: 1; transform: translateY(0); }
	}
	.animate-slide-up { animation: slide-up 0.2s ease-out both; }
</style>
