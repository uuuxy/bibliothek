<script>
	import { goto } from "$app/navigation";
	import { appState } from "$lib/store.svelte.js";
	import StrichcodeScanner from "$lib/components/StrichcodeScanner.svelte";

	let scannedResult = $state("");

	function onScanSuccess(decodedText) {
		scannedResult = decodedText;
		appState.searchQuery = decodedText;
		goto("/"); // eslint-disable-line svelte/no-navigation-without-resolve
	}
</script>

<div class="max-w-4xl mx-auto">
	<h1 class="text-2xl font-bold text-gray-900 mb-6">Scanner-Modus</h1>

	<div class="bg-white p-6 rounded-2xl shadow-sm border border-gray-200">
		<div
			class="aspect-video bg-black rounded-lg overflow-hidden mb-6 relative"
		>
			<StrichcodeScanner onCreated={onScanSuccess} />
		</div>

		{#if scannedResult}
			<div
				class="p-4 bg-emerald-50 text-emerald-700 rounded-lg border border-emerald-100"
			>
				Zuletzt gescannt: <span class="font-mono font-bold"
					>{scannedResult}</span
				>
			</div>
		{/if}
	</div>
</div>

