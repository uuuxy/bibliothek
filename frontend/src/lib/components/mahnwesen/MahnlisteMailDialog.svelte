<script>
	/**
	 * @component MahnlisteMailDialog
	 * Der Dialog „Mahnliste per E-Mail senden" — herausgelöst aus `MahnwesenTable`
	 * (05.09.2026), das mit ihm 244 Zeilen wog und damit über der 200-Zeilen-Regel
	 * lag. Eine Tabelle, die nebenbei einen Dialog mitbringt, ist zwei Dinge; der
	 * Zustand liegt ohnehin im Store, der Dialog braucht von der Tabelle nichts.
	 *
	 * Seit 07.09.2026 auf Modal.svelte; den Schließen-Knopf in der Kopfzeile stellt das
	 * Bauteil. Ebene „darueber": über der Mahnwesen-Tabelle mit ihren Menüs.
	 */
	import Modal from '../../Modal.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import { Mail } from '@lucide/svelte';
</script>

<Modal
	open={mahnwesenStore.modalOpen}
	onclose={mahnwesenStore.closeModal}
	ebene="darueber"
	beschriftetDurch="mahnliste-mail-titel"
>
	{#snippet header()}
		<h2 id="mahnliste-mail-titel" class="text-base font-bold text-on-surface">
			Mahnliste per E-Mail senden
		</h2>
	{/snippet}
	<div class="p-6 space-y-5">
		<div class="space-y-4">
			<div>
				<span class="block text-xs font-medium text-on-surface-variant mb-1">Klasse</span>
				<p class="text-sm font-semibold text-on-surface">{mahnwesenStore.modalKlasse}</p>
			</div>
			<Feld
				id="modal-email"
				label="E-Mail-Adresse des Klassenlehrers"
				type="email"
				bind:value={mahnwesenStore.modalEmail}
				placeholder="lehrer@schule.de"
				hint={mahnwesenStore.modalEmail.trim()
					? ''
					: 'Die Adresse wird aus dem Klassenlehrer-Mapping vorausgefüllt, kann aber geändert werden.'}
			/>
		</div>

		{#if mahnwesenStore.modalMsg}
			<div
				class="rounded-xl px-4 py-3 text-xs font-semibold {mahnwesenStore.modalMsg.type ===
				'success'
					? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
					: 'bg-rose-50 text-rose-600 border border-rose-200'}"
			>
				{mahnwesenStore.modalMsg.text}
			</div>
		{/if}

		<div class="flex justify-end gap-2">
			<Button variant="secondary" onclick={mahnwesenStore.closeModal}>Abbrechen</Button>
			<Button
				onclick={mahnwesenStore.sendMahnliste}
				disabled={mahnwesenStore.modalSending || mahnwesenStore.modalMsg?.type === 'success'}
			>
				{#if mahnwesenStore.modalSending}
					<Ladekreis size="sm" farbe="aktuell" />
				{:else}
					<Mail class="h-3.5 w-3.5" aria-hidden="true" />
				{/if}
				Senden
			</Button>
		</div>
	</div>
</Modal>
