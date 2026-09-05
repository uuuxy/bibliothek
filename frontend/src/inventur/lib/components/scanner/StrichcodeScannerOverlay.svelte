<script>
	import { fade } from 'svelte/transition';
	import KameraScanner from '$lib/components/scanner/KameraScanner.svelte';
	import { X } from '@lucide/svelte';
	import { escapeSchliesst } from '../../../../lib/components/ui/escapeSchliesst.js';

	let { isScanning = $bindable(), onScan } = $props();
	let scanStatus = $state('');

	/**
	 * @param {string} code
	 */
	function handleScan(code) {
		onScan(code);
		isScanning = false;
	}
</script>

{#if isScanning}
	<div
		class="fixed inset-0 z-60 flex items-center justify-center bg-black/80 backdrop-blur-sm p-4"
		transition:fade
	>
		<div
			class="bg-white p-6 rounded-3xl shadow-xl w-full max-w-md relative"
			use:escapeSchliesst={() => (isScanning = false)}
		>
			<button
				onclick={() => (isScanning = false)}
				class="absolute top-4 right-4 text-slate-500 hover:text-slate-800"
				aria-label="Scanner schließen"
			>
				<X class="w-6 h-6" aria-hidden="true" />
			</button>
			<h3 class="text-lg font-bold mb-4 text-center">ISBN scannen</h3>
			<KameraScanner
				onDecode={handleScan}
				onStatusChange={(/** @type {string} */ s) => (scanStatus = s)}
			/>
			<p class="text-center text-sm text-slate-600 mt-2">{scanStatus}</p>
		</div>
	</div>
{/if}
