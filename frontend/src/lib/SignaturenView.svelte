<script>
	import { onMount } from 'svelte';
	import { apiGet } from './apiFetch.js';
	import { authStore } from './stores/authStore.svelte.js';
	import { hatRecht } from './menu.js';
	import SignaturRegal from './components/signaturen/SignaturRegal.svelte';
	import SystematikVerwaltung from './components/signaturen/SystematikVerwaltung.svelte';
	import PageShell from './components/layout/PageShell.svelte';
	import Feld from './components/ui/Feld.svelte';

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

<PageShell>
	<p class="text-sm text-slate-500">
		Die Signatur ist die Regaladresse auf dem Buchrücken. Sie wird als Präfix gelesen: „BIB Deu“
		meint das ganze Regal, „BIB Deu 5 KRÜ“ ein einzelnes Fach darin.
	</p>

	<!-- Zwei Bereiche, keine zwei Kaesten: links die Liste, rechts das Regal dazu — in
	     M3 ein „supporting pane". Getrennt wird durch eine Haarlinie, senkrecht sobald
	     Platz ist, sonst waagerecht. Ein Rahmen mit Radius wuerde daraus zwei schwebende
	     Objekte machen; es ist aber EIN Arbeitsbereich mit zwei Haelften. -->
	<div class="grid divide-y divide-slate-200 lg:grid-cols-[20rem_1fr] lg:divide-x lg:divide-y-0">
		<section class="space-y-3 pb-6 lg:pr-6 lg:pb-0">
			<Feld id="sig-suche" label="Signatur suchen" bind:value={suche} placeholder="z. B. BIB" />

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
								<span class="text-sm text-slate-500 shrink-0">{sig.exemplare}</span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<section class="min-h-48 pt-6 lg:pt-0 lg:pl-6">
			<SignaturRegal signatur={gewaehlt} />
		</section>
	</div>

	{#if darfPflegen}
		<SystematikVerwaltung onChanged={ladeSignaturen} />
	{/if}
</PageShell>
