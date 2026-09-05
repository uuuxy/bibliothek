<script>
	import Button from './components/ui/Button.svelte';
	import Feld from './components/ui/Feld.svelte';
	import { escapeSchliesst } from './components/ui/escapeSchliesst.js';
	let { book, onCancel, onSubmit, isSubmitting } = $props();

	let damageReason = $state('Verloren');
	let damageAmount = $state(15.0);

	function handleSubmit() {
		onSubmit(damageReason, damageAmount);
	}
</script>

{#if book}
	<div class="fixed inset-0 z-60 flex items-center justify-center p-4">
		<div class="absolute inset-0 bg-slate-900/40 backdrop-blur-sm pointer-events-none"></div>
		<div
			class="bg-white rounded-3xl shadow-2xl p-6 max-w-md w-full relative z-10 animate-fade-in"
			use:escapeSchliesst={onCancel}
		>
			<h3 class="text-xl font-bold text-slate-800 mb-2">Verlust/Schaden melden</h3>
			<p class="text-sm text-slate-500 mb-4">
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
	</div>
{/if}
