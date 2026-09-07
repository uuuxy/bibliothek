<!-- @component LmfPlanUngespeichert — die Rückfrage, wenn jemand den Planer mit
     ungespeicherter Arbeit verlassen will (uiStore.blockierterWechsel). Zwei Ausgänge,
     keine dritte Option: bleiben und speichern, oder verwerfen und gehen. -->
<script>
	import Modal from '../../Modal.svelte';
	import Button from '../ui/Button.svelte';

	/** @type {{ offen: boolean, onbleiben: () => void, onverwerfen: () => void }} */
	let { offen, onbleiben, onverwerfen } = $props();
</script>

{#if offen}
	<Modal open={true} onclose={onbleiben} size="sm" beschriftetDurch="lmf-ungespeichert-titel">
		<div class="p-6">
			<h3 id="lmf-ungespeichert-titel" class="text-xl font-bold text-on-surface mb-2">
				Ungespeicherte Änderungen
			</h3>
			<p class="text-sm text-on-surface-variant mb-6">
				Der Plan wurde geändert, aber nicht gespeichert. Wer jetzt geht, verliert die Änderungen —
				gespeichert bleibt der letzte Stand.
			</p>
			<div class="flex justify-end gap-2">
				<Button variant="secondary" onclick={onbleiben}>Bleiben</Button>
				<Button variant="danger" onclick={onverwerfen}>Verwerfen und weiter</Button>
			</div>
		</div>
	</Modal>
{/if}
