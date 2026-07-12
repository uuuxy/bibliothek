<script>
	import { omniboxStore } from '../stores/omnibox.svelte.js';
</script>

{#if omniboxStore.vormerkungAlert}
	<div
		class="fixed inset-0 bg-rose-900/80 backdrop-blur-sm z-100 flex items-center justify-center p-4"
	>
		<div
			class="bg-white rounded-3xl p-8 max-w-md w-full text-center shadow-2xl border-4 border-rose-500"
		>
			<div class="text-6xl mb-4">🚨</div>
			<h2 class="text-2xl font-extrabold text-rose-700 mb-2">Achtung! Vorgemerkt!</h2>
			<p class="text-slate-700 mb-2">Dieses Medium wurde reserviert.</p>
			<p class="font-bold text-slate-900 mb-6">Achtung: Exemplar nicht ins Regal stellen!</p>
			{#if omniboxStore.vormerkungAlert.titel}
				<p class="text-sm text-slate-500 mb-2">„{omniboxStore.vormerkungAlert.titel}"</p>
			{/if}
			{#if omniboxStore.vormerkungAlert.user}
				<p
					class="text-md font-bold text-rose-800 bg-rose-100 py-3 px-4 rounded-xl border border-rose-200 mb-6"
				>
					Vorgemerkt für: {omniboxStore.vormerkungAlert.user}
				</p>
			{/if}
			<button
				onclick={() => {
					omniboxStore.vormerkungAlert = null;
				}}
				class="px-8 py-3 bg-rose-600 hover:bg-rose-700 text-white font-bold rounded-xl text-lg transition-colors cursor-pointer w-full"
			>
				Verstanden
			</button>
		</div>
	</div>
{/if}
