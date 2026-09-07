<!-- @component DamageReportModal — Verlust oder Schaden an einem entliehenen Buch melden.

     Seit 07.09.2026 auf Modal.svelte (Register 05.09.: elf Overlays bauten ihr Markup
     selbst). Ebene „darueber": Der Dialog öffnet aus der Schülerakte heraus, die selbst
     ein Overlay ist — auf der Grundebene läge er unsichtbar dahinter. -->
<script>
	import Modal from './Modal.svelte';
	import Button from './components/ui/Button.svelte';
	import Feld from './components/ui/Feld.svelte';
	let { book, onCancel, onSubmit, isSubmitting } = $props();

	let damageReason = $state('Verloren');
	let damageAmount = $state(15.0);

	function handleSubmit() {
		onSubmit(damageReason, damageAmount);
	}
</script>

{#if book}
	<Modal open={true} onclose={onCancel} ebene="darueber" beschriftetDurch="schaden-titel">
		<div class="p-6">
			<h3 id="schaden-titel" class="text-xl font-bold text-on-surface mb-2">
				Verlust/Schaden melden
			</h3>
			<p class="text-sm text-on-surface-variant mb-4">
				Für <strong>{book.titel}</strong> ({book.barcode_id}). Die Ausleihe wird beendet und eine
				Ersatzforderung an die Eltern generiert.
			</p>

			<div class="space-y-4">
				<Feld
					id="damage-reason"
					label="Grund"
					bind:value={damageReason}
					placeholder="z.B. Wasserschaden, Verloren..."
				/>
				<Feld
					id="damage-amount"
					label="Ersatzbetrag (€)"
					type="number"
					step="0.01"
					min="0"
					bind:value={damageAmount}
				/>
				<div class="flex gap-3 justify-end pt-4">
					<Button variant="ghost" onclick={onCancel} disabled={isSubmitting}>Abbrechen</Button>
					<Button
						variant="danger-solid"
						onclick={handleSubmit}
						disabled={isSubmitting || !damageReason.trim() || damageAmount < 0}
					>
						{isSubmitting ? 'Wird gemeldet...' : 'Melden & PDF generieren'}
					</Button>
				</div>
			</div>
		</div>
	</Modal>
{/if}
