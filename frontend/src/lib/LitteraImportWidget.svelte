<script lang="ts">
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
			const res = await fetch('/api/import/littera', {
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

<div class="littera-import-card">
	<h3>LITTERA CSV-Import (Altbestand)</h3>
	<p>Lade hier einen Export aus LITTERA als CSV-Datei (.csv) hoch. Erwartete Spalten: <em>Titel, Autor, Verlag, ISBN, Erscheinungsjahr, Kategorie/Systematik, Barcode/Exemplarnummer</em>.</p>

	<div class="upload-zone">
		<input 
			type="file" 
			accept=".csv" 
			bind:files 
			disabled={loading}
			class="file-input"
		/>
		
		<button 
			onclick={handleUpload} 
			disabled={loading || !files || files.length === 0}
			class="upload-btn"
		>
			{#if loading}
				<span class="spinner"></span> Importiere...
			{:else}
				Import Starten
			{/if}
		</button>
	</div>

	{#if resultMessage}
		<div class="result-message {isError ? 'error' : 'success'}">
			{resultMessage}
		</div>
	{/if}
</div>

<style>
	.littera-import-card {
		background: var(--surface);
		border-radius: 8px;
		padding: 24px;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
		border: 1px solid var(--border);
		margin-bottom: 24px;
	}

	h3 {
		margin-top: 0;
		color: var(--text-primary);
	}

	p {
		color: var(--text-secondary);
		font-size: 0.95rem;
		margin-bottom: 16px;
	}

	.upload-zone {
		display: flex;
		gap: 16px;
		align-items: center;
		margin-bottom: 16px;
	}

	.file-input {
		border: 1px dashed var(--border);
		padding: 12px;
		border-radius: 6px;
		flex-grow: 1;
		color: var(--text-primary);
	}

	.upload-btn {
		background: var(--primary);
		color: white;
		border: none;
		padding: 12px 24px;
		border-radius: 6px;
		font-weight: 600;
		cursor: pointer;
		display: flex;
		align-items: center;
		gap: 8px;
		transition: background 0.2s;
	}

	.upload-btn:hover:not(:disabled) {
		background: var(--primary-dark);
	}

	.upload-btn:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.spinner {
		width: 16px;
		height: 16px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: white;
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.result-message {
		padding: 12px;
		border-radius: 6px;
		font-weight: 500;
	}

	.success {
		background: rgba(46, 204, 113, 0.1);
		color: #27ae60;
		border: 1px solid rgba(46, 204, 113, 0.2);
	}

	.error {
		background: rgba(231, 76, 60, 0.1);
		color: #c0392b;
		border: 1px solid rgba(231, 76, 60, 0.2);
	}
</style>
