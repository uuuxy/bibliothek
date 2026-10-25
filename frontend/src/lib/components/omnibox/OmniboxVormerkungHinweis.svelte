<script>
	import { omniboxStore } from '../../stores/omnibox.svelte.js';
	import Button from '../ui/Button.svelte';
	import { escapeSchliesst } from '../ui/escapeSchliesst.js';
	import { fokusFalle } from '../ui/fokusFalle.js';
</script>

{#if omniboxStore.vormerkungHinweis}
	<!-- Der rote Schleier gehört zum Alarm: der dunkelste Ton der Fehlerfarbe statt des
	     schwarzen scrim. -->
	<div
		class="fixed inset-0 bg-scrim-alarm/80 backdrop-blur-sm z-100 flex items-center justify-center p-4"
	>
		<!-- alertdialog + Fokusfalle: Der Alarm unterbricht die Theke — der
		     Screenreader liest ihn sofort, Tab bleibt drin, Escape gibt den Fokus ans
		     Scanfeld zurück. -->
		<div
			role="alertdialog"
			aria-modal="true"
			aria-labelledby="omnibox-vormerkung-titel"
			class="bg-surface-container-lowest rounded-3xl p-8 max-w-md w-full text-center shadow-2xl border-4 border-error"
			use:fokusFalle
			use:escapeSchliesst={() => (omniboxStore.vormerkungHinweis = null)}
		>
			<div class="text-6xl mb-4">🚨</div>
			<h2 id="omnibox-vormerkung-titel" class="text-2xl font-extrabold text-error mb-2">
				Achtung! Vorgemerkt!
			</h2>
			<p class="text-on-surface-variant mb-2">Dieses Medium wurde reserviert.</p>
			<p class="font-bold text-on-surface mb-6">Achtung: Exemplar nicht ins Regal stellen!</p>
			{#if omniboxStore.vormerkungHinweis.titel}
				<p class="text-sm text-on-surface-variant mb-2">„{omniboxStore.vormerkungHinweis.titel}"</p>
			{/if}
			{#if omniboxStore.vormerkungHinweis.user}
				<p class="font-bold bg-error-container text-on-error-container py-3 px-4 rounded-xl mb-6">
					Vorgemerkt für: {omniboxStore.vormerkungHinweis.user}
				</p>
			{/if}
			<Button
				variant="danger-solid"
				size="lg"
				onclick={() => {
					omniboxStore.vormerkungHinweis = null;
				}}
				class="w-full text-lg"
			>
				Verstanden
			</Button>
		</div>
	</div>
{/if}
