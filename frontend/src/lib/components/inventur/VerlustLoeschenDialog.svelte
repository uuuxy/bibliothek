<!-- @component VerlustLoeschenDialog — Sicherheitsabfrage vorm endgültigen Löschen.

     Eigene, kleine Datei statt eines Inline-window.confirm(): Der Rest des Projekts
     hat native Browser-Dialoge konsequent durch eigene M3-Bauteile ersetzt (siehe die
     Auswahlfeld-Migration vom 04.08.), ein confirm() hier wäre ein Rückfall. Anders
     als der weiche Soft-Delete an anderer Stelle im Haus ist das hier ein echtes
     DELETE — die Abfrage muss die Zahl nennen, nicht nur "sicher?" fragen. -->
<script>
	import Modal from '../../Modal.svelte';
	import Button from '../ui/Button.svelte';

	/** @type {{ open: boolean, anzahl: number, laeuft: boolean, onConfirm: () => void, onClose: () => void }} */
	let { open, anzahl, laeuft, onConfirm, onClose } = $props();
</script>

<Modal {open} onclose={onClose} size="sm">
	{#snippet header()}
		<h3 class="text-sm font-bold text-slate-800">Endgültig löschen?</h3>
	{/snippet}
	<div class="p-6 space-y-4">
		<p class="text-sm text-slate-600">
			{anzahl}
			{anzahl === 1 ? 'Exemplar wird' : 'Exemplare werden'} unwiderruflich aus dem Katalog entfernt.
			Der Fehlbestandsbericht selbst bleibt erhalten — nur die Datensätze der Exemplare sind danach
			weg, das lässt sich nicht rückgängig machen.
		</p>
		<div class="flex justify-end gap-3 pt-2 border-t border-slate-100">
			<Button variant="secondary" onclick={onClose} disabled={laeuft}>Abbrechen</Button>
			<Button variant="danger-solid" onclick={onConfirm} disabled={laeuft}>
				{laeuft ? 'Wird gelöscht…' : `${anzahl} endgültig löschen`}
			</Button>
		</div>
	</div>
</Modal>
