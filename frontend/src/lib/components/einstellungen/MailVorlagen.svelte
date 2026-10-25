<!-- @component MailVorlagen — die Texte von Eltern-Mahnbrief und Bestellmail.

     Steht in der Einstellungs-Kategorie „Mail" unter deren Überschrift „Mail-Vorlagen", die
     auch den Hinweis trägt; eine eigene Überschrift hätte die Ordnung der Seite umgekehrt. -->
<script>
	import { TriangleAlert } from '@lucide/svelte';
	import { apiClient } from '../../apiFetch.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import MailVorlagenPlatzhalter from './MailVorlagenPlatzhalter.svelte';
	import { vorlagenName } from './mailVorlagenInfo.js';
	import { meldeFormular } from '../../stores/verlassenSchutz.svelte.js';

	/** @type {any[]} */
	let vorlagen = $state([]);
	let gewaehlteKennung = $state(null);
	let speichert = $state(false);
	let fehler = $state('');
	let laedt = $state(true);

	let gewaehlteVorlage = $derived(vorlagen.find((t) => t.id === gewaehlteKennung) || null);

	// Betreff und Text jeder Vorlage, wie sie gespeichert sind; nach dem Laden und je Vorlage nach
	// ihrem Speichern. Eine andere Vorlage wählen behält die Eingabe, die Kategorie verlassen nicht.
	/** @type {Record<string, { betreff: string, text_body: string }>} */
	let gespeichert = {};
	/** @param {any} t */
	const merke = (t) => (gespeichert[t.id] = { betreff: t.betreff, text_body: t.text_body });
	meldeFormular(() =>
		vorlagen.some((t) => {
			const g = gespeichert[t.id];
			return !!g && (t.betreff !== g.betreff || t.text_body !== g.text_body);
		})
	);

	$effect(() => {
		ladeVorlagen();
	});

	async function ladeVorlagen() {
		try {
			const res = await apiClient.get('/api/mail-templates');
			if (res.ok) {
				vorlagen = (await res.json()) || [];
				vorlagen.forEach(merke);
				if (vorlagen.length > 0 && !gewaehlteKennung) {
					gewaehlteKennung = vorlagen[0].id;
				}
			} else {
				fehler = 'Fehler beim Laden der Vorlagen.';
			}
		} catch (error) {
			console.error(error);
			fehler = 'Netzwerkfehler beim Laden.';
		} finally {
			laedt = false;
		}
	}

	async function speichereVorlage() {
		// Festgehalten vor dem Warten: Bis die Antwort kommt, kann eine andere Vorlage gewählt sein.
		const vorlage = gewaehlteVorlage;
		if (!vorlage) return;
		speichert = true;
		fehler = '';

		try {
			const res = await apiClient.put(`/api/mail-templates/${vorlage.id}`, {
				betreff: vorlage.betreff,
				text_body: vorlage.text_body
			});

			if (res.ok) {
				merke(vorlage);
				snackbars.addToast('Vorlage gespeichert.', 'success');
			} else {
				fehler = 'Fehler beim Speichern der Vorlage.';
			}
		} catch (error) {
			console.error(error);
			fehler = 'Netzwerkfehler beim Speichern.';
		} finally {
			speichert = false;
		}
	}

	/** @param {Event} e */
	function setzeBetreff(e) {
		if (!gewaehlteKennung) return;
		const val = /** @type {HTMLInputElement} */ (e.target).value;
		vorlagen = vorlagen.map((t) => (t.id === gewaehlteKennung ? { ...t, betreff: val } : t));
	}

	/** @param {Event} e */
	function setzeText(e) {
		if (!gewaehlteKennung) return;
		const val = /** @type {HTMLTextAreaElement} */ (e.target).value;
		vorlagen = vorlagen.map((t) => (t.id === gewaehlteKennung ? { ...t, text_body: val } : t));
	}
</script>

<section class="space-y-6">
	{#if fehler}
		<div
			role="alert"
			class="flex items-center gap-2 rounded-xl bg-error-container px-4 py-3 text-sm text-on-error-container"
		>
			<TriangleAlert class="h-4 w-4 shrink-0" aria-hidden="true" /><span>{fehler}</span>
		</div>
	{/if}

	<div class="flex flex-col lg:flex-row gap-10">
		<!-- Die Auswahl in der Form der Kategorienliste daneben (settings/KategorieListe). -->
		<div class="lg:w-1/3 flex flex-col gap-1">
			{#if laedt}
				<div class="py-4 text-sm text-center text-on-surface-variant">Lade Vorlagen...</div>
			{:else}
				{#each vorlagen as t, _i (_i)}
					{@const gewaehlt = gewaehlteKennung === t.id}
					<button
						type="button"
						aria-current={gewaehlt ? 'true' : undefined}
						class="flex flex-col rounded-2xl px-4 py-3 text-left transition-colors {gewaehlt
							? 'bg-secondary-container text-on-secondary-container'
							: 'text-on-surface-variant hover:bg-surface-container'}"
						onclick={() => {
							gewaehlteKennung = t.id;
							fehler = '';
						}}
					>
						<span class="truncate text-sm font-medium">{vorlagenName(t.typ)}</span>
						<span
							class="truncate text-sm {gewaehlt
								? 'text-on-secondary-container'
								: 'text-on-surface-variant'}">{t.betreff}</span
						>
					</button>
				{/each}
			{/if}
		</div>

		<div class="lg:w-2/3">
			{#if gewaehlteVorlage}
				<div class="flex flex-col h-full gap-6">
					<Feld
						id="betreff"
						etikett="Betreff"
						value={gewaehlteVorlage.betreff}
						oninput={setzeBetreff}
					/>

					<Feld
						id="text_body"
						etikett="Text-Inhalt"
						mehrzeilig
						zeilen={12}
						feld="font-mono resize-y"
						value={gewaehlteVorlage.text_body}
						oninput={setzeText}
					/>

					<MailVorlagenPlatzhalter typ={gewaehlteVorlage.typ} />

					<div class="flex justify-end pt-2">
						<Button size="lg" onclick={speichereVorlage} disabled={speichert} class="px-6">
							{speichert ? 'Speichern...' : 'Vorlage Speichern'}
						</Button>
					</div>
				</div>
			{:else if vorlagen.length > 0}
				<div
					class="h-full flex flex-col items-center justify-center p-8 border-2 border-dashed border-outline-variant rounded-xl text-on-surface-variant"
				>
					<p class="text-sm">Bitte wählen Sie links eine Vorlage aus.</p>
				</div>
			{/if}
		</div>
	</div>
</section>
