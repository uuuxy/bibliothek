<!-- @component PapierkorbLoeschenDialog — Rückfrage vorm endgültigen Löschen (Art. 17).

     Anders als das Archivieren (Soft-Delete, nächtliche Frist räumt nach) ist das hier
     der sofortige, unumkehrbare Löschweg: PurgeStudent tilgt Ausleihhistorie und
     Audit-Spur und entfernt den Datensatz. Eigener M3-Dialog statt window.confirm(),
     aus demselben Grund wie beim Fehlbestand (VerlustLoeschenDialog). -->
<script>
	import Modal from '../../Modal.svelte';
	import Button from '../ui/Button.svelte';

	/** @type {{ open: boolean, name: string, laeuft: boolean, onConfirm: () => void, onClose: () => void }} */
	let { open, name, laeuft, onConfirm, onClose } = $props();
</script>

<Modal {open} onclose={onClose} size="sm">
	{#snippet header()}
		<h3 class="text-base font-bold text-on-surface">Endgültig löschen?</h3>
	{/snippet}
	<div class="p-6 space-y-4">
		<p class="text-sm text-on-surface-variant">
			<span class="font-semibold text-on-surface">{name}</span> wird sofort, endgültig und unwiderruflich
			gelöscht — samt Ausleihhistorie und Protokollspur (Art. 17 DSGVO). Das ist der Weg für ein Löschverlangen
			der Eltern; ohne Verlangen räumt die nächtliche Frist den Papierkorb von selbst.
		</p>
		<div class="flex justify-end gap-3 pt-2 border-t border-outline-variant">
			<Button variant="secondary" onclick={onClose} disabled={laeuft}>Abbrechen</Button>
			<Button variant="danger-solid" onclick={onConfirm} disabled={laeuft}>
				{laeuft ? 'Wird gelöscht…' : 'Endgültig löschen'}
			</Button>
		</div>
	</div>
</Modal>
