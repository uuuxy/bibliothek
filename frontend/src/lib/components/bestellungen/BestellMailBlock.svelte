<!-- @component BestellMailBlock — der gescheiterte Versand der Bestellmail an der Bestellung.
     Aufbau wie BestellStatusBlock: ein Zustand, eine Aussage, eine Aktion. Er steht, solange
     der Server den gescheiterten Versand an der Bestellung führt (mail_gescheitert_am);
     „Erneut senden" schickt die Mail aus der gespeicherten Bestellung noch einmal. -->
<script>
	import { apiPost, FRIST_MAILVERSAND_MS } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import { formatZeitpunkt } from '../../utils/format.js';
	import Button from '../ui/Button.svelte';
	import { MailWarning } from '@lucide/svelte';

	/** @type {{ b: any, darfSenden: boolean, onAktualisieren: () => Promise<void> }} */
	let { b, darfSenden, onAktualisieren } = $props();

	let laeuft = $state(false);

	async function erneutSenden() {
		laeuft = true;
		try {
			// Der Server verschickt die Mail in der Anfrage, deshalb die Frist der Mail-Aufrufe.
			const antwort = await apiPost(`/api/bestellungen/${b.id}/mail`, null, {
				timeoutMs: FRIST_MAILVERSAND_MS
			});
			toastStore.addToast(
				antwort?.message ?? 'Bestellmail gesendet.',
				antwort?.status === 'warning' ? 'error' : 'success'
			);
		} catch {
			// Die Meldung des Servers hat apiFetch schon gezeigt.
		} finally {
			// Auch nach einem gescheiterten Versuch neu laden: Der Zeitpunkt an der Bestellung
			// ist dann der des neuen Versuchs.
			await onAktualisieren();
			laeuft = false;
		}
	}
</script>

<!-- Getönter Container in den Fehler-Rollen: M3 nimmt sie für Fehlerzustände, die
     Container-Rolle für die Fläche dahinter. -->
<div
	class="mb-3 flex flex-wrap items-center justify-between gap-3 rounded-xl bg-error-container px-4 py-3 text-on-error-container"
>
	<div class="flex items-start gap-3">
		<MailWarning size={20} class="mt-0.5 shrink-0" aria-hidden="true" />
		<div>
			<p class="text-sm font-semibold">Die Bestellmail ist nicht rausgegangen</p>
			<p class="mt-0.5 text-xs">
				Letzter Versuch: {formatZeitpunkt(b.mail_gescheitert_am)}. Der Händler hat die Bestellung
				nicht erhalten.
			</p>
		</div>
	</div>
	{#if darfSenden}
		<Button variant="secondary" size="sm" disabled={laeuft} onclick={erneutSenden}>
			Erneut senden
		</Button>
	{/if}
</div>
