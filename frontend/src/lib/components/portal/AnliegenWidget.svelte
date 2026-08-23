<!-- @component AnliegenWidget — Wünsche und Meldungen der Lehrkraft im
     Kollegiums-Portal (Betreiber-Entscheidung 18.08.2026): Wünschen geht
     IMMER, ohne Stichtag. Formular oben, eigene Anliegen mit Status darunter —
     „die LMF hakt ab, und du siehst es hier oder bekommst eine Mail".

     Seit dem 23.08.2026 ein eigener REITER des Portals statt eines Anhängsels
     unter der Buchsuche. Vorher standen zwei ungleiche Aufgaben auf einer Fläche:
     oben ein namenloses Suchfeld, darunter ein 340-px-Poster, darunter dieses
     Formular — und weil die Felder dieselbe Pillenform trugen wie die Suche,
     las sich „Welches Buch?" wie ein zweites Suchfeld.

     Die Felder sind deshalb jetzt SettingField (Beschriftung über dem Feld,
     Rahmen statt Füllung). Die Regel im Haus: Pille = suchen, Rahmen = eingeben. -->
<script>
	import { apiFetch } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import SettingField from '../settings/SettingField.svelte';

	/** @typedef {{ id: string, art: string, titel_text: string, klasse: string, kommentar?: string, erstellt_am: string, erledigt_am?: string, erledigt_notiz?: string }} Anliegen */

	let art = $state('wunsch');
	let titelText = $state('');
	let klasse = $state('');
	let kommentar = $state('');
	let sending = $state(false);

	// Die Liste gehört dem Portal: Es braucht sie ohnehin für den Zähler am Reiter und
	// für die Startfläche. Zwei eigene Abrufe hätten zwei Wahrheiten über denselben
	// Zustand ergeben — nach dem Absenden hätte der Zähler noch den alten Stand gezeigt.
	/** @type {{ anliegen: Anliegen[], onaktualisiert: () => void | Promise<void> }} */
	let { anliegen, onaktualisiert } = $props();
	const eigene = $derived(anliegen);

	async function absenden() {
		if (!titelText.trim() || sending) return;
		sending = true;
		try {
			const res = await apiFetch('/api/anliegen', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					art,
					titel_text: titelText.trim(),
					klasse: klasse.trim(),
					kommentar: kommentar.trim()
				})
			});
			if (!res.ok) {
				const data = await res.json().catch(() => null);
				throw new Error(data?.error || 'Anliegen konnte nicht gesendet werden.');
			}
			toastStore.addToast(
				art === 'wunsch' ? 'Wunsch ist bei der Bibliothek.' : 'Meldung ist bei der Bibliothek.',
				'success'
			);
			titelText = '';
			klasse = '';
			kommentar = '';
			await onaktualisiert();
		} catch (err) {
			toastStore.addToast(/** @type {any} */ (err).message || String(err), 'error');
		} finally {
			sending = false;
		}
	}
</script>

<section class="flex w-full max-w-3xl flex-col gap-6">
	<p class="text-sm text-on-surface-variant">
		Buchwunsch für deine Klasse oder etwas stimmt nicht? Die Bibliothek arbeitet die Liste ab — beim
		Erledigen bekommst du eine Mail.
	</p>

	<div class="flex flex-col gap-4">
		<!-- Zwei Arten, ein Formular: der Unterschied ist nur das Etikett. -->
		<div class="flex gap-2" role="radiogroup" aria-label="Art des Anliegens">
			{#each [['wunsch', 'Buchwunsch'], ['meldung', 'Etwas stimmt nicht']] as [wert, label] (wert)}
				<button
					role="radio"
					aria-checked={art === wert}
					onclick={() => (art = wert)}
					class="px-4 py-2 rounded-full text-sm font-semibold transition-colors cursor-pointer {art ===
					wert
						? 'bg-primary-container text-on-primary-container'
						: 'bg-surface border border-outline-variant text-on-surface-variant hover:bg-surface-container-low'}"
				>
					{label}
				</button>
			{/each}
		</div>

		<SettingField
			bind:value={titelText}
			label={art === 'wunsch' ? 'Welches Buch?' : 'Worum geht es?'}
			type="text"
			maxlength={300}
			placeholder={art === 'wunsch'
				? 'z. B. Markl Biologie 2, ISBN falls bekannt'
				: 'z. B. 8G3 hat die falschen Bücher bekommen'}
		/>
		<div class="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-3">
			<SettingField
				bind:value={klasse}
				label="Klasse / Kurs"
				type="text"
				maxlength={50}
				placeholder="z. B. 8G3"
			/>
			<SettingField
				bind:value={kommentar}
				label="Anmerkung (optional)"
				type="text"
				maxlength={1000}
				placeholder="Was die Bibliothek sonst noch wissen sollte"
				class="sm:col-span-2"
			/>
		</div>
		<div class="flex justify-end">
			<Button onclick={absenden} disabled={sending || !titelText.trim()}>
				{sending ? 'Wird gesendet …' : 'Absenden'}
			</Button>
		</div>
	</div>

	{#if eigene.length > 0}
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
