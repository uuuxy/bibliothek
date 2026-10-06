<script>
	import { apiClient } from './apiFetch.js';
	import { bestaetigen } from './stores/bestaetigung.svelte.js';
	import { toastStore } from './stores/toastStore.svelte.js';
	import { onMount } from 'svelte';
	import Button from './components/ui/Button.svelte';
	import Feld from './components/ui/Feld.svelte';
	import Select from './components/ui/Select.svelte';
	import {
		erzeugeKlassenVorschlaege,
		klassenPlatzhalter
	} from './components/students/klassenVorschlaege.svelte.js';

	/** @type {string} */
	let klasse = $state('');
	/** @type {string} */
	let neuesDatum = $state('');
	/** @type {boolean} */
	let isExtending = $state(false);
	// Auswählen statt tippen (docs/OFFEN.md 5.18, 30.09.2026): die Klassen, in denen Schüler
	// sind. Ein Tippfehler legte hier keine Klasse an, verlängerte aber still nichts.
	const klassenListe = erzeugeKlassenVorschlaege();
	const klassen = $derived(klassenListe.liste);
	onMount(klassenListe.lade);

	async function handleGlobalExtend() {
		if (!klasse.trim() || !neuesDatum) {
			toastStore.addToast('Bitte Klasse wählen und neues Rückgabedatum eingeben.', 'warning');
			return;
		}

		const confirmed = await bestaetigen({
			titel: `Alle LMF-Ausleihen der Klasse ${klasse} verlängern?`,
			text: `Neues Rückgabedatum ${neuesDatum}. Das verändert möglicherweise hunderte Datensätze gleichzeitig.`,
			aktion: 'Verlängern',
			gefaehrlich: true
		});
		if (!confirmed) return;

		isExtending = true;
		try {
			const res = await apiClient.post('/api/ausleihen/global-extend-lmf', {
				klasse: klasse.trim(),
				neues_rueckgabe_datum: neuesDatum
			});

			if (res.ok) {
				const data = await res.json();
				toastStore.addToast(`${data.updated_count} Ausleihen wurden verlängert.`, 'success');
				klasse = '';
				neuesDatum = '';
			} else {
				const errText = await res.text();
				toastStore.addToast(`Fehler: ${errText}`, 'error');
			}
		} catch (e) {
			console.error(e);
			toastStore.addToast('Netzwerkfehler beim Senden der Anfrage.', 'error');
		} finally {
			isExtending = false;
		}
	}
</script>

<!-- Ohne PageShell: Das ist keine eigene Route, sondern Inhalt der Einstellungs-Kategorie
     „LMF-Aktionen" — das Seitengerüst stellt SystemSettings. -->
<div class="space-y-5">
	<div>
		<h3 class="text-base font-bold text-on-surface">LMF-Massenverlängerung (Klasse)</h3>
		<p class="text-xs text-on-surface-variant mt-1 leading-relaxed max-w-lg">
			Verlängert alle aktiven LMF-Ausleihen (Schulbücher) einer bestimmten Klasse auf ein neues
			fixes Rückgabedatum.
		</p>
	</div>

	<div class="flex items-end gap-4 flex-wrap">
		<div class="grid gap-y-1.5">
			<label for="extendKlasse" class="text-sm font-medium text-on-surface-variant">Klasse</label>
			<Select
				id="extendKlasse"
				bind:value={klasse}
				options={klassen.map((k) => ({ value: k, label: k }))}
				placeholder={klassenPlatzhalter(klassen.length, klassenListe.ladefehler)}
				class="w-36"
			/>
		</div>

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
