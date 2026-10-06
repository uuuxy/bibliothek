<!-- @component VerlustLoeschenDialog — Sicherheitsabfrage vorm endgültigen Löschen.

     Eigene, kleine Datei statt eines window.confirm(): Das Haus führt keine nativen
     Browser-Dialoge. Es ist ein echtes DELETE und kein Papierkorb — die Abfrage nennt die
     Zahl, statt nur „sicher?" zu fragen. -->
<script>
	import Modal from '../../Modal.svelte';
	import Button from '../ui/Button.svelte';

	/** @type {{ open: boolean, anzahl: number, laeuft: boolean, onConfirm: () => void, onClose: () => void }} */
	let { open, anzahl, laeuft, onConfirm, onClose } = $props();
</script>

<Modal {open} onclose={onClose} size="sm">
	{#snippet header()}
		<h3 class="text-base font-bold text-on-surface">Endgültig löschen?</h3>
	{/snippet}
	<div class="p-6 space-y-4">
		<p class="text-sm text-on-surface-variant">
			{anzahl}
			{anzahl === 1 ? 'Exemplar wird' : 'Exemplare werden'} unwiderruflich aus dem Katalog entfernt. Der
			Fehlbestandsbericht selbst bleibt erhalten — nur die Datensätze der Exemplare sind danach weg, das
			lässt sich nicht rückgängig machen.
		</p>
		<div class="flex justify-end gap-3 pt-2 border-t border-outline-variant">
			<Button variant="secondary" onclick={onClose} disabled={laeuft}>Abbrechen</Button>
			<Button variant="danger-solid" onclick={onConfirm} disabled={laeuft}>
				{laeuft ? 'Wird gelöscht…' : `${anzahl} endgültig löschen`}
			</Button>
		</div>
	</div>
</Modal>
