<!-- @component AnliegenWidget — Wünsche und Meldungen der Lehrkraft im
     Kollegiums-Portal (Betreiber-Entscheidung 18.08.2026): Wünschen geht
     IMMER, ohne Stichtag. Formular oben, eigene Anliegen mit Status darunter —
     „die LMF hakt ab, und du siehst es hier oder bekommst eine Mail". -->
<script>
	import { onMount } from 'svelte';
	import { apiFetch } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import Button from '../ui/Button.svelte';

	/** @typedef {{ id: string, art: string, titel_text: string, klasse: string, kommentar?: string, erstellt_am: string, erledigt_am?: string, erledigt_notiz?: string }} Anliegen */

	let art = $state('wunsch');
	let titelText = $state('');
	let klasse = $state('');
	let kommentar = $state('');
	let sending = $state(false);

	/** @type {Anliegen[]} */
	let eigene = $state([]);

	async function ladeEigene() {
		try {
			const res = await apiFetch('/api/anliegen/eigene');
			eigene = res.ok ? await res.json() : [];
		} catch {
			eigene = [];
		}
	}

	onMount(ladeEigene);

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
			await ladeEigene();
		} catch (err) {
			toastStore.addToast(/** @type {any} */ (err).message || String(err), 'error');
		} finally {
			sending = false;
		}
	}
</script>

<section class="w-full mt-10">
	<h2 class="text-base font-bold text-on-surface">Wünsche & Meldungen</h2>
	<p class="text-sm text-on-surface-variant mt-0.5 mb-4">
		Buchwunsch für deine Klasse oder etwas stimmt nicht? Die Bibliothek arbeitet die Liste ab — beim
		Erledigen bekommst du eine Mail.
	</p>

	<div class="space-y-3">
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

		<input
			type="text"
			bind:value={titelText}
			maxlength="300"
			placeholder={art === 'wunsch'
				? 'Welches Buch? (z. B. „Markl Biologie 2, ISBN falls bekannt“)'
				: 'Worum geht es? (z. B. „8G3 hat die falschen Bücher bekommen“)'}
			class="w-full text-sm border border-outline-variant rounded-full px-4 bg-surface"
		/>
		<div class="flex gap-3">
			<input
				type="text"
				bind:value={klasse}
				maxlength="50"
				placeholder="Klasse / Kurs (z. B. 8G3)"
				class="w-40 text-sm border border-outline-variant rounded-full px-4 bg-surface"
			/>
			<input
				type="text"
				bind:value={kommentar}
				maxlength="1000"
				placeholder="Anmerkung (optional)"
				class="flex-1 text-sm border border-outline-variant rounded-full px-4 bg-surface"
			/>
			<Button
				variant="primary"
				size="sm"
				onclick={absenden}
				disabled={sending || !titelText.trim()}
			>
				{sending ? 'Sende…' : 'Absenden'}
			</Button>
		</div>
	</div>

	{#if eigene.length > 0}
		<ul class="divide-y divide-outline-variant mt-6">
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
	{/if}
</section>
