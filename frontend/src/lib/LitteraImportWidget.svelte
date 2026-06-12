<script lang="ts">
	import { apiFetch } from './apiFetch.js';

	let files: FileList | null = $state(null);
	let loading = $state(false);
	let resultMessage = $state('');
	let isError = $state(false);

	async function handleUpload() {
		if (!files || files.length === 0) return;

		loading = true;
		resultMessage = '';
		isError = false;

		const formData = new FormData();
		formData.append('file', files[0]);

		try {
			const res = await apiFetch('/api/import/littera', {
				method: 'POST',
				body: formData
			});

			const data = await res.json();

			if (!res.ok) {
				throw new Error(data.error || 'Upload fehlgeschlagen');
			}

			resultMessage = `Erfolgreich importiert! Neue Titel: ${data.new_titles_count}, Neue Exemplare: ${data.imported_copies_count}.`;
			files = null; // reset input
		} catch (err: any) {
			isError = true;
			resultMessage = err.message || 'Ein unerwarteter Fehler ist aufgetreten.';
		} finally {
			loading = false;
		}
	}
</script>

<div class="p-6 rounded-3xl bg-white border border-slate-100 shadow-xs space-y-4">
	<div>
		<h3 class="text-base font-bold text-slate-900">LITTERA CSV-Import (Altbestand)</h3>
		<p class="text-xs text-slate-500 mt-1">Lade hier einen Export aus LITTERA als CSV-Datei (.csv) hoch. Erwartete Spalten: <em>Titel, Autor, Verlag, ISBN, Erscheinungsjahr, Kategorie/Systematik, Barcode/Exemplarnummer</em>.</p>
	</div>

	<div class="flex items-center gap-4">
		<input 
			type="file" 
			accept=".csv" 
			bind:files 
			disabled={loading}
			class="flex-1 bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-sm focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none text-slate-800"
		/>
		
		<button 
			onclick={handleUpload} 
			disabled={loading || !files || files.length === 0}
			class="px-6 py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-bold text-sm rounded-xl transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
		>
			{#if loading}
				<div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
				<span>Importiere...</span>
			{:else}
				<span>Import Starten</span>
			{/if}
		</button>
	</div>

	{#if resultMessage}
		<div class="p-4 rounded-xl text-sm font-semibold {isError ? 'bg-rose-50 text-rose-600 border border-rose-100' : 'bg-emerald-50 text-emerald-700 border border-emerald-100'}">
			{resultMessage}
		</div>
	{/if}
</div>
