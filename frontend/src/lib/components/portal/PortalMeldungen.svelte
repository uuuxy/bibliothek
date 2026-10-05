<!-- @component PortalMeldungen — die eigenen Meldungen der Lehrkraft mit ihrem Stand und
     darunter „Problem melden" ohne Buch. Steht unter der Suche des Portals, solange nichts
     gesucht wird; am Treffer gibt es dieselbe Meldung mit gewähltem Buch (PortalTrefferkarte). -->
<script>
	import { tick } from 'svelte';
	import { TriangleAlert } from '@lucide/svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import Button from '../ui/Button.svelte';
	import ProblemFormular from './ProblemFormular.svelte';

	/** @typedef {{ id: string, art: string, titel_text: string, klasse: string, kommentar?: string, erstellt_am: string, erledigt_am?: string, erledigt_notiz?: string }} Anliegen */

	// Liste und Formularzustand gehören dem Portal (eigeneAnliegen, problemMeldung): Nach
	// dem Absenden am Treffer liest es dieselbe Liste neu.
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

<section class="flex w-full max-w-3xl flex-col gap-3">
	<!-- Ein gescheiterter erster Abruf sagt nichts über die Meldungen — dann steht hier der
	     Ausfall und nicht die leere Liste. „Nichts da" ließe eine abgeschickte Meldung als
	     verloren erscheinen, und der nächste Schritt wäre, sie noch einmal zu schicken. -->
	{#if ladefehler}
		<LadeFehler
			onerneut={onaktualisiert}
			titel="Deine Meldungen konnten nicht geladen werden"
			text="Bitte später noch einmal versuchen. Schon abgeschickte Meldungen sind nicht verloren — sie sind bei der Bibliothek."
		/>
	{:else if eigene.length > 0}
		<h2 class="text-base font-medium text-on-surface">Deine Meldungen</h2>
		<ul class="divide-y divide-outline-variant">
			{#each eigene as a (a.id)}
				<li class="flex items-start justify-between gap-4 py-2.5">
					<div class="min-w-0 flex-1">
						<p class="truncate text-sm text-on-surface">
							<span class="font-semibold">{a.art === 'wunsch' ? 'Wunsch' : 'Meldung'}:</span>
							{a.titel_text}
							{#if a.klasse}<span class="text-on-surface-variant">· {a.klasse}</span>{/if}
						</p>
						{#if a.erledigt_am && a.erledigt_notiz}
							<p class="mt-0.5 text-xs text-on-surface-variant italic">
								Bibliothek: „{a.erledigt_notiz}"
							</p>
						{/if}
					</div>
					<span
						class="inline-flex shrink-0 items-center rounded-full px-2 py-0.5 text-label-small font-semibold {a.erledigt_am
							? 'bg-secondary-container text-on-secondary-container'
							: 'border border-outline-variant bg-surface text-on-surface-variant'}"
					>
						{a.erledigt_am ? 'Erledigt' : 'Offen'}
					</span>
				</li>
			{/each}
		</ul>
	{/if}

	{#if form.open}
		<div class="pt-3">
			<ProblemFormular {form} mitWorum onsenden={senden} onabbrechen={abbrechen} />
		</div>
	{:else}
		<p class="pt-3 text-sm text-on-surface-variant">
			Etwas stimmt nicht? Die Bibliothek arbeitet die Liste ab — beim Erledigen bekommst du eine
			Mail.
		</p>
		<div>
			<Button variant="secondary" size="lg" bind:element={knopf} onclick={onoeffnen}>
				<TriangleAlert class="h-5 w-5" aria-hidden="true" />
				Problem melden
			</Button>
		</div>
	{/if}
</section>
