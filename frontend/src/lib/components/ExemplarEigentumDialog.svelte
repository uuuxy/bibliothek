<!-- @component ExemplarEigentumDialog — das Eigentum markierter Exemplare ändern
     (docs/OFFEN.md 4.24, Stufe 3, freigegeben am 29.09.2026). Wie in Littera „Exemplardaten
     anpassen": markieren, einen Wert für alle setzen.

     Bauform wie „Topf der Bestellung ändern" (bestellungen/BestellMittelDialog): Modal,
     Radio-Gruppe, Pflicht-Grund, die Aktion erst mit Auswahl und Grund frei. M3 Dialogs:
     „Disable confirming actions until a choice is made", höchstens zwei Aktionen. M3 Radio
     button: „a single selection from a list of options"; im Dialog wirkt sie erst beim
     Bestätigen. Keine Auswahl vorbelegt: Die markierten Exemplare können verschiedene Werte
     tragen, und eine Vorbelegung wäre eine Entscheidung, die niemand getroffen hat. -->
<script>
	import Modal from '../Modal.svelte';
	import Button from './ui/Button.svelte';
	import Feld from './ui/Feld.svelte';
	import Radio from './ui/Radio.svelte';
	import { apiPut } from '../apiFetch.js';
	import { toastStore } from '../stores/toastStore.svelte.js';
	import { MITTEL, MITTEL_REIHENFOLGE } from './bestellungen/mittel.js';

	/**
	 * @type {{ open: boolean, exemplarIds: string[], onclose: () => void, onGeaendert: () => Promise<void> | void }}
	 */
	let { open, exemplarIds, onclose, onGeaendert } = $props();

	/** Nichts gewählt: ''. „Vorgabe" ist ein eigener Wert, damit „nichts gewählt" frei bleibt. */
	const VORGABE = 'vorgabe';
	let gewaehlt = $state('');
	let grund = $state('');
	let sendet = $state(false);

	$effect(() => {
		if (open) {
			gewaehlt = '';
			grund = '';
		}
	});

	const anzahl = $derived(exemplarIds.length);
	const bereit = $derived(gewaehlt !== '' && grund.trim() !== '' && !sendet);

	async function speichern() {
		if (!bereit) return;
		sendet = true;
		try {
			const antwort = await apiPut('/api/exemplare/eigentum', {
				exemplar_ids: exemplarIds,
				eigentum: gewaehlt === VORGABE ? '' : gewaehlt,
				grund: grund.trim()
			});
			const n = antwort?.geaendert ?? 0;
			toastStore.addToast(
				n === 0 ? 'Eigentum war schon so eingetragen.' : `Eigentum geändert: ${n} Exemplar(e).`,
				'success'
			);
			await onGeaendert();
			onclose();
		} catch {
			/* apiFetch zeigt den Fehler-Toast */
		} finally {
			sendet = false;
		}
	}
</script>

<Modal {open} {onclose} size="sm" beschriftetDurch="eigentum-dialog-titel">
	<div class="space-y-4 p-6">
		<h2 id="eigentum-dialog-titel" class="text-lg font-bold text-on-surface">Eigentum ändern</h2>
		<p class="text-sm leading-relaxed text-on-surface-variant">
			Gilt für {anzahl}
			{anzahl === 1 ? 'Exemplar' : 'Exemplare'}. Wirkt auf Etikett, Bestandsbücher und
			Schadensersatz. Die Änderung wird mit Grund protokolliert.
		</p>

		<fieldset class="space-y-2">
			<legend class="text-xs font-medium text-on-surface-variant">Eigentum</legend>
			{#each MITTEL_REIHENFOLGE as m (m)}
				<label class="flex items-center gap-3 text-sm text-on-surface">
					<Radio bind:group={gewaehlt} value={m} aria-label={MITTEL[m].traeger} />
					<span
						>{MITTEL[m].traeger}
						<span class="text-on-surface-variant">({MITTEL[m].label})</span></span
					>
				</label>
			{/each}
			<label class="flex items-center gap-3 text-sm text-on-surface">
				<Radio bind:group={gewaehlt} value={VORGABE} aria-label="Vorgabe" />
				<span>Vorgabe <span class="text-on-surface-variant">(aus Bestellung oder Titel)</span></span
				>
			</label>
		</fieldset>

		<Feld
			id="eigentum-grund"
			label="Grund der Änderung"
			bind:value={grund}
			placeholder="z. B. Klassensatz aus LMF-Mitteln gekauft"
			required
		/>

		<div class="flex justify-end gap-2 pt-2">
			<Button variant="ghost" onclick={onclose} disabled={sendet}>Abbrechen</Button>
			<Button onclick={speichern} disabled={!bereit}>
				{sendet ? 'Wird gespeichert …' : 'Eigentum ändern'}
			</Button>
		</div>
	</div>
</Modal>
