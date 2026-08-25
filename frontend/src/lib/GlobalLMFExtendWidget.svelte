<script>
	import { apiFetch, apiClient } from './apiFetch.js';
	import { onMount } from 'svelte';
	import Button from './components/ui/Button.svelte';
	import Feld from './components/ui/Feld.svelte';

	/** @type {string} */
	let klasse = $state('');
	/** @type {string} */
	let neuesDatum = $state('');
	/** @type {boolean} */
	let isExtending = $state(false);
	/** @type {string[]} Vorschläge für das Klassenfeld (freies Tippen bleibt möglich) */
	let klassenVorschlaege = $state([]);

	onMount(async () => {
		// Echte Schüler-Klassen als Vorschläge (GET /api/klassen liefert DISTINCT klasse).
		// Rein optional: schlägt der Abruf fehl (z. B. fehlendes view_students-Recht),
		// tippt man den Klassennamen einfach frei ein.
		try {
			const res = await apiFetch('/api/klassen', { credentials: 'include' });
			if (res.ok) {
				const data = await res.json();
				klassenVorschlaege = Array.isArray(data) ? data : [];
			}
		} catch {
			/* Vorschläge sind optional */
		}
	});

	async function handleGlobalExtend() {
		if (!klasse.trim() || !neuesDatum) {
			alert('Bitte Klasse und neues Rückgabedatum eingeben.');
			return;
		}

		const confirmed = confirm(
			`ACHTUNG: Möchten Sie wirklich alle LMF-Ausleihen der Klasse ${klasse} auf den ${neuesDatum} verlängern?\nDies verändert möglicherweise hunderte Datensätze gleichzeitig!`
		);
		if (!confirmed) return;

		isExtending = true;
		try {
			const res = await apiClient.post('/api/ausleihen/global-extend-lmf', {
				klasse: klasse.trim(),
				neues_rueckgabe_datum: neuesDatum
			});

			if (res.ok) {
				const data = await res.json();
				alert(`Erfolgreich: ${data.updated_count} Ausleihen wurden verlängert!`);
				klasse = '';
				neuesDatum = '';
			} else {
				const errText = await res.text();
				alert(`Fehler: ${errText}`);
			}
		} catch (e) {
			console.error(e);
			alert('Netzwerkfehler beim Senden der Anfrage.');
		} finally {
			isExtending = false;
		}
	}
</script>

<!-- Ohne PageShell: Seit dem 24.08.2026 keine eigene Route mehr, sondern Inhalt der
     Einstellungs-Kategorie „LMF-Aktionen" — das Seitengerüst stellt SystemSettings. -->
<div class="space-y-5">
	<div>
		<h3 class="text-base font-bold text-slate-900">LMF-Massenverlängerung (Klasse)</h3>
		<p class="text-xs text-slate-500 mt-1 leading-relaxed max-w-lg">
			Verlängert alle aktiven LMF-Ausleihen (Schulbücher) einer bestimmten Klasse auf ein neues
			fixes Rückgabedatum.
		</p>
	</div>

	<div class="flex items-end gap-4 flex-wrap">
		<Feld
			id="extendKlasse"
			label="Klasse (z.B. 10b)"
			list="lmf-klassen-vorschlaege"
			bind:value={klasse}
			placeholder="10b"
			class="w-32"
		/>
		<datalist id="lmf-klassen-vorschlaege">
			{#each klassenVorschlaege as k (k)}
				<option value={k}></option>
			{/each}
		</datalist>

		<Feld
			id="extendDatum"
			label="Neues Rückgabedatum"
			type="date"
			bind:value={neuesDatum}
			class="w-48"
		/>

		<Button
			onclick={handleGlobalExtend}
			disabled={isExtending || !klasse.trim() || !neuesDatum}
			class="px-6"
		>
			{isExtending ? 'Wird verarbeitet...' : 'Klassen-LMF global verlängern'}
		</Button>
	</div>
</div>
