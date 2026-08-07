<script>
	import { onMount } from 'svelte';
	import { apiGet } from './apiFetch.js';
	import { authStore } from './stores/authStore.svelte.js';
	import { hatRecht } from './menu.js';
	import SignaturRegal from './components/signaturen/SignaturRegal.svelte';
	import SystematikVerwaltung from './components/signaturen/SystematikVerwaltung.svelte';
	import PageShell from './components/layout/PageShell.svelte';

	let signaturen = $state(/** @type {any[]} */ ([]));
	let laedt = $state(true);
	let suche = $state('');
	let gewaehlt = $state('');

	const darfPflegen = $derived(hatRecht(authStore.currentUser, 'edit_books'));

	const gefiltert = $derived(
		signaturen.filter((s) => s.signatur.toLowerCase().includes(suche.trim().toLowerCase()))
	);

	async function ladeSignaturen() {
		laedt = true;
		try {
			signaturen = (await apiGet('/api/signaturen')) || [];
		} catch {
			// apiGet hat die Servermeldung bereits als Toast gezeigt.
		} finally {
			laedt = false;
		}
	}

	onMount(ladeSignaturen);
</script>

<PageShell
	breite="voll"
	titel="Signaturen"
	beschreibung="Die Regaladresse auf dem Buchrücken — als Präfix gelesen."
>
	<div>
		<h1 class="text-2xl font-bold text-slate-900">Signaturen</h1>
		<p class="text-sm text-slate-500 mt-1">
			Die Signatur ist die Regaladresse auf dem Buchrücken. Sie wird als Präfix gelesen: „BIB Deu“
			meint das ganze Regal, „BIB Deu 5 KRÜ“ ein einzelnes Fach darin.
		</p>
	</div>

	<div class="grid gap-6 lg:grid-cols-[20rem_1fr] items-start">
		<section class="bg-white border border-slate-200 rounded-xl p-4 space-y-3">
			<label for="sig-suche" class="block text-sm font-medium text-slate-700">
				Signatur suchen
			</label>
			<input
				id="sig-suche"
				bind:value={suche}
				placeholder="z. B. BIB"
				class="h-9 w-full rounded-lg border border-slate-300 bg-slate-50 px-3 text-sm text-slate-900 outline-none focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/20"
			/>

			{#if laedt}
				<p class="text-sm text-slate-500">Wird geladen …</p>
			{:else if signaturen.length === 0}
				<p class="text-sm text-slate-500">
					Noch kein Buch trägt eine Signatur. Sie entsteht am Buch selbst — im Buchformular,
					vorgeschlagen aus den Sachgruppen.
				</p>
			{:else if gefiltert.length === 0}
				<p class="text-sm text-slate-500">Keine Signatur passt zu „{suche}“.</p>
			{:else}
				<ul class="max-h-112 overflow-y-auto -mx-1">
					{#each gefiltert as sig (sig.signatur)}
						<li>
							<button
								type="button"
								onclick={() => (gewaehlt = sig.signatur)}
								class="w-full text-left px-3 py-2 rounded-lg flex items-baseline justify-between gap-2 transition-colors {gewaehlt ===
								sig.signatur
									? 'bg-blue-50 text-blue-900'
									: 'hover:bg-slate-50 text-slate-700'}"
							>
								<span class="font-mono text-sm truncate">{sig.signatur}</span>
								<span class="text-xs text-slate-500 shrink-0">{sig.exemplare}</span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<section class="bg-white border border-slate-200 rounded-xl p-5 min-h-48">
			<SignaturRegal signatur={gewaehlt} />
		</section>
	</div>

	{#if darfPflegen}
		<SystematikVerwaltung onChanged={ladeSignaturen} />
	{/if}
</PageShell>
