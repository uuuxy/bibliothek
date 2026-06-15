<script lang="ts">
	import { apiFetch } from "./apiFetch.js";

	let files: FileList | null = $state(null);
	let isImporting = $state(false);
	let importResult: { type: 'success' | 'error', message: string } | null = $state(null);

	async function handleUpload() {
		if (!files || files.length === 0) return;

		isImporting = true;
		importResult = null;

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

			let successMsg = "";
			if (data.type === "xml") {
				successMsg = `XML-Import erfolgreich! ${data.updated_titles_count} bestehende Titel wurden aktualisiert.`;
			} else {
				successMsg = `CSV-Import erfolgreich! ${data.new_titles_count} neue Titel und ${data.imported_copies_count} Exemplare wurden angelegt.`;
			}

			importResult = {
				type: 'success',
				message: successMsg
			};
			files = null; // reset input
		} catch (err: any) {
			importResult = {
				type: 'error',
				message: err.message || 'Ein unerwarteter Fehler ist aufgetreten.'
			};
		} finally {
			isImporting = false;
		}
	}
</script>

<div class="p-6 rounded-3xl bg-white border border-slate-100 shadow-xs space-y-4">
	<div>
		<h3 class="text-base font-bold text-slate-900">Katalog-Import (Littera)</h3>
		<p class="text-xs text-slate-500 mt-1">Lade hier die <strong>katalogisat.xml</strong> hoch (um bestehende Buch-Metadaten zu aktualisieren) oder eine <strong>CSV-Datei</strong> (um neue Bücher/Exemplare per Bulk-Insert anzulegen).</p>
	</div>

	<div class="flex items-center gap-4">
		<label class="relative {isImporting ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}">
			<input 
				type="file" 
				accept=".xml,.csv" 
				bind:files 
				disabled={isImporting}
				class="sr-only"
			/>
			<div class="px-5 py-2.5 bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold text-sm rounded-xl transition-colors border border-slate-200 inline-block">
				{files && files.length > 0 ? files[0].name : 'Datei auswählen...'}
			</div>
		</label>
		
		<button 
			onclick={handleUpload} 
			disabled={isImporting || !files || files.length === 0}
			class="px-6 py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-bold text-sm rounded-xl transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
		>
			{#if isImporting}
				<div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
				<span>Importiere...</span>
			{:else}
				<span>Import Starten</span>
			{/if}
		</button>
	</div>

	{#if importResult}
		<div class="p-4 rounded-xl text-sm font-semibold {importResult.type === 'error' ? 'bg-rose-50 text-rose-600 border border-rose-100' : 'bg-emerald-50 text-emerald-700 border border-emerald-100'}">
			{importResult.message}
		</div>
	{/if}
</div>
