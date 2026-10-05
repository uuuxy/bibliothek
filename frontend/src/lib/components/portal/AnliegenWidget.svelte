<!-- @component AnliegenWidget — Wünsche und Meldungen der Lehrkraft im Kollegiums-Portal.

     Der Einstieg sind zwei Knöpfe ohne Vorbelegung: Ein Wunsch bleibt liegen, bis bestellt
     wird, ein Problem soll am selben Tag erledigt werden, und mit einer vorbelegten Art
     stünde das eine als das andere in der Liste der Bibliothek. Darunter stehen die eigenen
     Anliegen mit ihrem Stand. -->
<script>
	import { tick } from 'svelte';
	import { BookPlus, TriangleAlert } from '@lucide/svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import Button from '../ui/Button.svelte';
	import AnliegenFormular from './AnliegenFormular.svelte';

	/** @typedef {{ id: string, art: string, titel_text: string, klasse: string, kommentar?: string, erstellt_am: string, erledigt_am?: string, erledigt_notiz?: string }} Anliegen */

	// Die Liste gehört dem Portal: Es braucht sie ohnehin für den Zähler am Reiter und
	// für die Startfläche. Zwei eigene Abrufe hätten zwei Wahrheiten über denselben
	// Zustand ergeben — nach dem Absenden hätte der Zähler noch den alten Stand gezeigt.
	/** @type {{ anliegen: Anliegen[], onaktualisiert: () => void | Promise<void>, ladefehler?: boolean }} */
	let { anliegen, onaktualisiert, ladefehler = false } = $props();
	const eigene = $derived(anliegen);

	/** @type {'wunsch' | 'meldung' | null} */
	let art = $state(null);
	/** @type {HTMLButtonElement | undefined} */
	let wunschKnopf = $state();
	/** @type {HTMLButtonElement | undefined} */
	let meldungKnopf = $state();

	/** Zurück zur Wahl; der Fokus geht auf den Knopf, der das Formular geöffnet hatte. */
	async function zurWahl() {
		const zuletzt = art;
		art = null;
		await tick();
		(zuletzt === 'meldung' ? meldungKnopf : wunschKnopf)?.focus();
	}

	async function abgeschickt() {
		await zurWahl();
		await onaktualisiert();
	}
</script>

<section class="flex w-full max-w-3xl flex-col gap-6">
	<p class="text-sm text-on-surface-variant">
		Buchwunsch für deine Klasse oder etwas stimmt nicht? Die Bibliothek arbeitet die Liste ab — beim
		Erledigen bekommst du eine Mail.
	</p>

	{#if art === null}
		<!-- M3 Buttons: zwei gleichrangige Wahlen in derselben Form, das Symbol vor dem Wort. -->
		<div class="flex flex-wrap gap-3" role="group" aria-label="Art des Anliegens">
			<Button
				variant="secondary"
				size="lg"
				bind:element={wunschKnopf}
				onclick={() => (art = 'wunsch')}
			>
				<BookPlus class="h-5 w-5" aria-hidden="true" />
				Buchwunsch
			</Button>
			<Button
				variant="secondary"
				size="lg"
				bind:element={meldungKnopf}
				onclick={() => (art = 'meldung')}
			>
				<TriangleAlert class="h-5 w-5" aria-hidden="true" />
				Problem melden
			</Button>
		</div>
	{:else}
		<AnliegenFormular {art} onabbrechen={zurWahl} onabgeschickt={abgeschickt} />
	{/if}

	<!-- Ein gescheiterter ERSTER Abruf sagt nichts über die Anliegen — dann steht hier der
	     Ausfall und nicht die leere Liste. „Nichts da" hätte einen abgeschickten Wunsch als
	     verloren erscheinen lassen, und der nächste Schritt wäre gewesen, ihn noch einmal
	     zu schicken. -->
	{#if ladefehler}
		<div class="border-t border-outline-variant pt-6">
			<LadeFehler
				onerneut={onaktualisiert}
				titel="Deine Anliegen konnten nicht geladen werden"
				text="Bitte später noch einmal versuchen. Schon abgeschickte Wünsche und Meldungen sind nicht verloren — sie sind bei der Bibliothek."
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
