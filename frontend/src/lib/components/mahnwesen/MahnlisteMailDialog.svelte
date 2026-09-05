<script>
	/**
	 * @component MahnlisteMailDialog
	 * Der Dialog „Mahnliste per E-Mail senden" — herausgelöst aus `MahnwesenTable`
	 * (05.09.2026), das mit ihm 244 Zeilen wog und damit über der 200-Zeilen-Regel
	 * lag. Eine Tabelle, die nebenbei einen Dialog mitbringt, ist zwei Dinge; der
	 * Zustand liegt ohnehin im Store, der Dialog braucht von der Tabelle nichts.
	 */
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import { escapeSchliesst } from '../ui/escapeSchliesst.js';
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import { X, Mail } from '@lucide/svelte';
</script>

{#if mahnwesenStore.modalOpen}
	<div class="fixed inset-0 z-60 flex items-center justify-center p-4">
		<div
			class="absolute inset-0 bg-black/20 backdrop-blur-sm"
			onclick={mahnwesenStore.closeModal}
			aria-hidden="true"
		></div>
		<div
			class="relative bg-white rounded-3xl shadow-2xl w-full max-w-md p-6 space-y-5"
			use:escapeSchliesst={mahnwesenStore.closeModal}
		>
			<div class="flex items-center justify-between">
				<h2 class="text-base font-bold text-slate-800">Mahnliste per E-Mail senden</h2>
				<button
					onclick={mahnwesenStore.closeModal}
					aria-label="Modal schließen"
					class="p-1.5 rounded-lg text-slate-400 hover:bg-slate-100 transition-colors"
				>
					<X class="h-4 w-4" aria-hidden="true" />
				</button>
			</div>

			<div class="space-y-4">
				<div>
					<span class="block text-xs font-medium text-slate-500 mb-1">Klasse</span>
					<p class="text-sm font-semibold text-slate-800">{mahnwesenStore.modalKlasse}</p>
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
						<div
							class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"
						></div>
					{:else}
						<Mail class="h-3.5 w-3.5" aria-hidden="true" />
					{/if}
					Senden
				</Button>
			</div>
		</div>
	</div>
{/if}
