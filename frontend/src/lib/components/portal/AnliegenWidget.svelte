<!-- @component AnliegenWidget — Meldungen der Lehrkraft im Kollegiums-Portal: „Problem
     melden" ohne Buch und darunter die eigenen Anliegen mit ihrem Stand. Am Treffer der
     Suche gibt es dieselbe Meldung mit gewähltem Buch (PortalTrefferkarte). -->
<script>
	import { tick } from 'svelte';
	import { TriangleAlert } from '@lucide/svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import Button from '../ui/Button.svelte';
	import ProblemFormular from './ProblemFormular.svelte';

	/** @typedef {{ id: string, art: string, titel_text: string, klasse: string, kommentar?: string, erstellt_am: string, erledigt_am?: string, erledigt_notiz?: string }} Anliegen */

	// Liste und Formularzustand gehören dem Portal (eigeneAnliegen, problemMeldung): Der
	// Zähler am Reiter liest dieselbe Liste, und zwei Abrufe wären zwei Wahrheiten.
	/** @type {{ anliegen: Anliegen[], form: import('./problemMeldung.svelte.js').MeldeFormular, onoeffnen: () => void, onsenden: () => Promise<boolean>, onabbrechen: () => void, onaktualisiert: () => void | Promise<void>, ladefehler?: boolean }} */
	let {
		anliegen,
		form,
		onoeffnen,
		onsenden,
		onabbrechen,
		onaktualisiert,
		ladefehler = false
	} = $props();
	const eigene = $derived(anliegen);

	/** @type {HTMLButtonElement | undefined} */
	let knopf = $state();

	// Der Knopf steht nur, solange das Formular zu ist; danach geht der Fokus auf ihn zurück.
	async function zumKnopf() {
		await tick();
		knopf?.focus();
	}
	async function abbrechen() {
		onabbrechen();
		await zumKnopf();
	}
	async function senden() {
		if (await onsenden()) await zumKnopf();
	}
</script>

<section class="flex w-full max-w-3xl flex-col gap-6">
	<p class="text-sm text-on-surface-variant">
		Etwas stimmt nicht? Die Bibliothek arbeitet die Liste ab — beim Erledigen bekommst du eine Mail.
	</p>

	{#if form.open}
		<ProblemFormular {form} mitWorum onsenden={senden} onabbrechen={abbrechen} />
	{:else}
		<div>
			<Button variant="secondary" size="lg" bind:element={knopf} onclick={onoeffnen}>
				<TriangleAlert class="h-5 w-5" aria-hidden="true" />
				Problem melden
			</Button>
		</div>
	{/if}

	<!-- Ein gescheiterter erster Abruf sagt nichts über die Anliegen — dann steht hier der
	     Ausfall und nicht die leere Liste. „Nichts da" ließe eine abgeschickte Meldung als
	     verloren erscheinen, und der nächste Schritt wäre, sie noch einmal zu schicken. -->
	{#if ladefehler}
		<div class="border-t border-outline-variant pt-6">
			<LadeFehler
				onerneut={onaktualisiert}
				titel="Deine Anliegen konnten nicht geladen werden"
				text="Bitte später noch einmal versuchen. Schon abgeschickte Meldungen sind nicht verloren — sie sind bei der Bibliothek."
			/>
		</div>
	{:else if eigene.length > 0}
		<div class="flex flex-col gap-2 border-t border-outline-variant pt-6">
			<h3 class="text-base font-medium text-on-surface">Deine Anliegen</h3>
			<ul class="divide-y divide-outline-variant">
				{#each eigene as a (a.id)}
					<li class="py-3 flex items-start justify-between gap-4">
						<div class="min-w-0 flex-1">
							<p class="text-sm text-on-surface truncate">
								<span class="font-semibold">{a.art === 'wunsch' ? 'Wunsch' : 'Meldung'}:</span>
								{a.titel_text}
								{#if a.klasse}<span class="text-on-surface-variant">· {a.klasse}</span>{/if}
							</p>
							{#if a.erledigt_am && a.erledigt_notiz}
								<p class="text-xs text-on-surface-variant italic mt-0.5">
									Bibliothek: „{a.erledigt_notiz}"
								</p>
							{/if}
						</div>
						<span
							class="shrink-0 inline-flex items-center px-2 py-0.5 rounded-full text-label-small font-semibold {a.erledigt_am
								? 'bg-secondary-container text-on-secondary-container'
								: 'bg-surface border border-outline-variant text-on-surface-variant'}"
						>
							{a.erledigt_am ? 'Erledigt' : 'Offen'}
						</span>
					</li>
				{/each}
			</ul>
		</div>
	{/if}
</section>
