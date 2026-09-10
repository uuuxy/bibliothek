<!-- @component Der Rückweg für den Topf einer Bestellung — Dialog mit Auswahl und
     Pflicht-Grund.

     Eine Bestellung im falschen Topf ließ sich bis zum 10.09.2026 nicht korrigieren; für
     die Berichte, die nach Topf rechnen, stünde sie dauerhaft im falschen Block, und
     Alt-Bestellungen „ohne Zuordnung" blieben es für immer. Der Dialog ändert NUR die
     eigene Zuordnung: Das Anschreiben, das der Händler bekommen hat, bleibt, wie es
     rausging. Deshalb ist der Grund Pflicht, und der Eingriff steht im Admin-Audit-Log.

     Bauform wie der Storno-Dialog der Gebühren (StudentGebuehrenCard): Modal, Radio-Gruppe,
     ein Feld für den Grund, die Aktion erst mit Grund freigegeben. -->
<script>
	import Modal from '../../Modal.svelte';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import Radio from '../ui/Radio.svelte';
	import { apiPut } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import { MITTEL, MITTEL_REIHENFOLGE, mittelLabel } from './mittel.js';

	/**
	 * @type {{
	 *   open: boolean,
	 *   bestellungId: string,
	 *   aktuell: string,
	 *   onclose: () => void,
	 *   onAktualisieren: () => Promise<void> | void
	 * }}
	 * aktuell — der heutige Topf ('' = ohne Zuordnung); onAktualisieren lädt die Bestellung neu.
	 */
	let { open, bestellungId, aktuell, onclose, onAktualisieren } = $props();

	/** @type {string} */
	let gewaehlt = $state('');
	let grund = $state('');
	let sendet = $state(false);

	// Beim Öffnen auf den Stand der Bestellung setzen — nicht beim Einhängen: Der Dialog
	// bleibt eingehängt und wird nur ein- und ausgeblendet.
	$effect(() => {
		if (open) {
			gewaehlt = aktuell;
			grund = '';
		}
	});

	const unveraendert = $derived(gewaehlt === aktuell);
	const bereit = $derived(!unveraendert && grund.trim() !== '' && !sendet);

	async function speichern() {
		if (!bereit) return;
		sendet = true;
		try {
			await apiPut(`/api/bestellungen/${bestellungId}/mittel`, {
				mittel: gewaehlt,
				grund: grund.trim()
			});
			toastStore.addToast(`Topf geändert: ${mittelLabel(gewaehlt)}.`, 'success');
			await onAktualisieren();
			onclose();
		} catch {
			/* apiFetch zeigt den Fehler-Toast */
		} finally {
			sendet = false;
		}
	}
</script>

<Modal {open} {onclose} size="sm" beschriftetDurch="mittel-dialog-titel">
	<div class="space-y-4 p-6">
		<h2 id="mittel-dialog-titel" class="text-lg font-bold text-on-surface">
			Topf der Bestellung ändern
		</h2>
		<p class="text-sm leading-relaxed text-on-surface-variant">
			Ändert nur die Zuordnung in dieser Anwendung. Das Anschreiben, das der Händler bekommen hat,
			bleibt unverändert. Die Änderung wird mit Grund protokolliert.
		</p>

		<fieldset class="space-y-2">
			<legend class="text-xs font-medium text-on-surface-variant">Bezahlt aus</legend>
			{#each MITTEL_REIHENFOLGE as m (m)}
				<label class="flex items-center gap-3 text-sm text-on-surface">
					<Radio bind:group={gewaehlt} value={m} aria-label={mittelLabel(m)} />
					<span
						>{MITTEL[m].label}
						<span class="text-on-surface-variant">({MITTEL[m].traeger})</span></span
					>
				</label>
			{/each}
		</fieldset>

		<Feld
			id="mittel-grund"
			label="Grund der Korrektur"
			bind:value={grund}
			placeholder="z. B. Titel war falsch als Lernmittel gekennzeichnet"
			required
		/>

		<div class="flex justify-end gap-2 pt-2">
			<Button variant="ghost" onclick={onclose} disabled={sendet}>Abbrechen</Button>
			<Button onclick={speichern} disabled={!bereit}>
				{sendet ? 'Wird gespeichert …' : 'Topf ändern'}
			</Button>
		</div>
	</div>
</Modal>
