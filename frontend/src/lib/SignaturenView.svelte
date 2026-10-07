<script>
	import { onMount } from 'svelte';
	import { apiGet } from './apiFetch.js';
	import { authStore } from './stores/authStore.svelte.js';
	import { hatRecht } from './menu.js';
	import SignaturRegal from './components/signaturen/SignaturRegal.svelte';
	import SystematikVerwaltung from './components/signaturen/SystematikVerwaltung.svelte';
	import PageShell from './components/layout/PageShell.svelte';
	import Suchpille from './components/ui/Suchpille.svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';

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
	<!-- Zwei Bereiche, keine zwei Kästen: links die Liste, rechts das Regal dazu — in M3 ein
	     „supporting pane". Getrennt wird durch eine Haarlinie, senkrecht sobald Platz ist,
	     sonst waagerecht; ein Rahmen mit Radius machte daraus zwei schwebende Objekte. -->
	<!-- Die Suche steht über beiden Hälften: Sie ist die eine Suche der Seite und hat
	     dieselbe Breite und Kante wie überall sonst. Der Erklärsatz steht unter dem Feld
	     (in M3 der „supporting text"), damit die Pille auf der Startlinie bleibt. -->
	<div class="flex flex-col gap-2">
		<Suchpille
			id="signaturen-suchfeld"
			bind:wert={suche}
			etikett="Signatur suchen"
			platzhalter="Signatur suchen"
		/>
		<p class="text-sm text-on-surface-variant">
			Die Signatur ist die Regaladresse auf dem Buchrücken. Sie wird als Präfix gelesen: „LMF Deu 7“
			meint das ganze Regal, „LMF Deu 7 / Bie“ ein einzelnes Buch darin.
		</p>
	</div>

	<div
		class="grid divide-y divide-outline-variant lg:grid-cols-[20rem_minmax(0,1fr)] lg:divide-x lg:divide-y-0"
	>
		<section class="space-y-3 pb-6 lg:pr-6 lg:pb-0">
			{#if laedt}
				<div class="flex justify-center py-6"><Ladekreis label="Signaturen laden" /></div>
			{:else if signaturen.length === 0}
				<p class="text-sm text-on-surface-variant">
					Noch kein Buch trägt eine Signatur. Sie entsteht am Buch selbst — im Buchformular,
					vorgeschlagen aus den Sachgruppen.
				</p>
			{:else if gefiltert.length === 0}
				<p class="text-sm text-on-surface-variant">Keine Signatur passt zu „{suche}“.</p>
			{:else}
				<ul class="max-h-112 overflow-y-auto -mx-1">
					{#each gefiltert as sig (sig.signatur)}
						<li>
							<button
								type="button"
								onclick={() => (gewaehlt = sig.signatur)}
								class="w-full text-left px-3 py-2 rounded-lg flex items-baseline justify-between gap-2 transition-colors {gewaehlt ===
								sig.signatur
									? 'bg-secondary-container text-on-secondary-container'
									: 'text-on-surface'}"
							>
								<span class="font-mono text-sm truncate">{sig.signatur}</span>
								<span class="text-sm text-on-surface-variant shrink-0">{sig.exemplare}</span>
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
