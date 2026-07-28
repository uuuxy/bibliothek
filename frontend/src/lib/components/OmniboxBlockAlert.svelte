<script>
	import { omniboxStore } from '../stores/omnibox.svelte.js';
	import { apiClient } from '../apiFetch.js';
	import Button from './ui/Button.svelte';

	/** @type {{ onReload: () => void }} */
	let { onReload } = $props();
</script>

{#if omniboxStore.blockAlert}
	<div
		class="fixed inset-0 bg-rose-900/80 backdrop-blur-sm z-100 flex items-center justify-center p-4"
	>
		<div
			class="bg-white rounded-3xl p-8 max-w-md w-full text-center shadow-2xl border-4 border-rose-500"
		>
			<div class="text-6xl mb-4">⛔️</div>
			<h2 class="text-2xl font-extrabold text-rose-700 mb-2">Ausleihe blockiert</h2>
			<p class="text-slate-700 font-medium mb-6">{omniboxStore.blockAlert.message}</p>

			<div class="space-y-3">
				<Button
					variant="danger-solid"
					size="lg"
					onclick={() => {
						const q = omniboxStore.blockAlert?.query;
						if (!q) return;
						omniboxStore.blockAlert = null;
						omniboxStore.queryVal = q;
						omniboxStore.submitAction(null, onReload, true);
					}}
					class="w-full text-lg"
				>
					Einmalig ignorieren (Override)
				</Button>

				{#if omniboxStore.activeStudent?.is_manually_blocked}
					<Button
						variant="secondary"
						size="lg"
						onclick={async () => {
							try {
								const res = await apiClient.patch(
									`/api/admin/students/${omniboxStore.activeStudent.id}/lock`,
									{
										is_locked: false
									}
								);
								if (res.ok) {
									const q = omniboxStore.blockAlert?.query;
									omniboxStore.blockAlert = null;
									if (q) omniboxStore.queryVal = q;
									omniboxStore.activeStudent.is_manually_blocked = false;
									omniboxStore.submitAction(null, onReload);
								}
							} catch (e) {
								console.error(e);
							}
						}}
						class="w-full text-lg"
					>
						Sperre dauerhaft aufheben
					</Button>
				{/if}

				<Button
					variant="ghost"
					size="lg"
					onclick={() => {
						omniboxStore.blockAlert = null;
					}}
					class="mt-2 w-full"
				>
					Abbrechen
				</Button>
			</div>
		</div>
	</div>
{/if}
