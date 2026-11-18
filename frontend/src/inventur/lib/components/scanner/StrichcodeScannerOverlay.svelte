<script>
	import { fade } from 'svelte/transition';
	import { emphasizedAccelerate, emphasizedDecelerate } from '../../../../lib/utils/bewegung.js';
	import KameraScanner from '$lib/components/scanner/KameraScanner.svelte';
	import Icon from '../../../../lib/components/ui/Icon.svelte';
	import { escapeSchliesst } from '../../../../lib/components/ui/escapeSchliesst.js';

	let { isScanning = $bindable(), onScan } = $props();
	let scanStatus = $state('');

	/**
	 * @param {string} code
	 */
	function beiScan(code) {
		onScan(code);
		isScanning = false;
	}
</script>

{#if isScanning}
	<div
		class="fixed inset-0 z-60 flex items-center justify-center bg-scrim/80 backdrop-blur-sm p-4"
		in:fade={{ duration: 400, easing: emphasizedDecelerate }}
		out:fade={{ duration: 200, easing: emphasizedAccelerate }}
	>
		<div
			class="bg-surface-container-lowest p-6 rounded-3xl shadow-level3 w-full max-w-md relative"
			use:escapeSchliesst={() => (isScanning = false)}
		>
			<button
				onclick={() => (isScanning = false)}
				class="icon-btn absolute top-4 right-4 text-on-surface-variant"
				aria-label="Scanner schließen"
				data-tip="Scanner schließen"
			>
				<Icon name="close" groesse={24} />
			</button>
			<h3 class="text-lg mb-4 text-center">ISBN scannen</h3>
			<KameraScanner
				onDecode={beiScan}
				onStatusChange={(/** @type {string} */ s) => (scanStatus = s)}
			/>
			<p class="text-center text-sm text-on-surface-variant mt-2">{scanStatus}</p>
		</div>
	</div>
{/if}
