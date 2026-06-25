<script>
	import { apiClient } from "../../../lib/apiFetch.js";
	import { onMount } from 'svelte';
	import KlassenlehrerMapping from "../../lib/components/settings/KlassenlehrerMapping.svelte";

	let loading = $state(true);
	let saving = $state(false);
	let toast = $state(/** @type {{msg:string,type:string}|null} */ (null));

	let ferienLeseclubAktiv = $state(false);
	let ferienLeseclubZieldatum = $state('');
	let lmfStichtag = $state('07-31');

	/**
	 * @param {string} msg
	 * @param {string} [type]
	 */
	function showToast(msg, type = 'success') {
		toast = { msg, type };
		setTimeout(() => { toast = null; }, 3500);
	}

	onMount(async () => {
		try {
			const res = await apiClient.get('/api/einstellungen');
			if (res.ok) {
				const data = await res.json();
				ferienLeseclubAktiv = data.ferien_leseclub_aktiv ?? false;
				ferienLeseclubZieldatum = data.ferien_leseclub_zieldatum ?? '';
				lmfStichtag = data.lmf_stichtag ?? '07-31';
			}
		} catch { /* use defaults */ }
		loading = false;
	});

	async function saveSettings() {
		saving = true;
		try {
			const res = await apiClient.post('/api/einstellungen', {
					ferien_leseclub_aktiv: ferienLeseclubAktiv,
					ferien_leseclub_zieldatum: ferienLeseclubZieldatum || null,
					lmf_stichtag: lmfStichtag || '07-31'
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
		<div class="py-6 border-b border-gray-200 space-y-5">
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
					aria-checked={ferienLeseclubAktiv}
					aria-label="Ferien-Leseclub aktivieren">
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
		<div class="py-6 border-b border-gray-200 space-y-4">
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
					class="mt-1 block w-40 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none" />
				<p class="text-xs text-gray-400 mt-1">Format: MM-TT (z. B. <code>07-31</code> für 31. Juli)</p>
			</label>
		</div>

		<!-- ── Fristen-Übersicht ── -->
		<div class="py-6 border-b border-gray-200">
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
		<KlassenlehrerMapping {showToast} />

		<!-- ── System ── -->
		<div class="py-6 border-b border-gray-200 space-y-4">
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
