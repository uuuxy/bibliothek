<!-- @component MailTemplates — die Texte von Eltern-Mahnbrief und Bestellmail.

     Steht in der Einstellungs-Kategorie „Mail" unter deren Überschrift „Mail-Vorlagen", die
     auch den Hinweis trägt; eine eigene Überschrift hätte die Ordnung der Seite umgekehrt. -->
<script>
	import { TriangleAlert } from '@lucide/svelte';
	import { apiClient } from './apiFetch.js';
	import { toastStore } from './stores/toastStore.svelte.js';
	import Button from './components/ui/Button.svelte';
	import Feld from './components/ui/Feld.svelte';
	import MailVorlagenPlatzhalter from './MailVorlagenPlatzhalter.svelte';
	import { vorlagenName } from './mailVorlagenInfo.js';

	/** @type {any[]} */
	let templates = $state([]);
	let selectedTemplateId = $state(null);
	let isSaving = $state(false);
	let errorMessage = $state('');
	let laedt = $state(true);

	let selectedTemplate = $derived(templates.find((t) => t.id === selectedTemplateId) || null);

	$effect(() => {
		loadTemplates();
	});

	async function loadTemplates() {
		try {
			const res = await apiClient.get('/api/mail-templates');
			if (res.ok) {
				templates = (await res.json()) || [];
				if (templates.length > 0 && !selectedTemplateId) {
					selectedTemplateId = templates[0].id;
				}
			} else {
				errorMessage = 'Fehler beim Laden der Vorlagen.';
			}
		} catch (error) {
			console.error(error);
			errorMessage = 'Netzwerkfehler beim Laden.';
		} finally {
			laedt = false;
		}
	}

	async function saveTemplate() {
		if (!selectedTemplate) return;
		isSaving = true;
		errorMessage = '';

		try {
			const res = await apiClient.put(`/api/mail-templates/${selectedTemplate.id}`, {
				betreff: selectedTemplate.betreff,
				text_body: selectedTemplate.text_body
			});

			if (res.ok) {
				toastStore.addToast('Vorlage gespeichert.', 'success');
			} else {
				errorMessage = 'Fehler beim Speichern der Vorlage.';
			}
		} catch (error) {
			console.error(error);
			errorMessage = 'Netzwerkfehler beim Speichern.';
		} finally {
			isSaving = false;
		}
	}

	/** @param {Event} e */
	function updateBetreff(e) {
		if (!selectedTemplateId) return;
		const val = /** @type {HTMLInputElement} */ (e.target).value;
		templates = templates.map((t) => (t.id === selectedTemplateId ? { ...t, betreff: val } : t));
	}

	/** @param {Event} e */
	function updateTextBody(e) {
		if (!selectedTemplateId) return;
		const val = /** @type {HTMLTextAreaElement} */ (e.target).value;
		templates = templates.map((t) => (t.id === selectedTemplateId ? { ...t, text_body: val } : t));
	}
</script>

<section class="space-y-6">
	{#if errorMessage}
		<div
			role="alert"
			class="flex items-center gap-2 rounded-xl bg-error-container px-4 py-3 text-sm text-on-error-container"
		>
			<TriangleAlert class="h-4 w-4 shrink-0" aria-hidden="true" /><span>{errorMessage}</span>
		</div>
	{/if}

	<div class="flex flex-col lg:flex-row gap-10">
		<!-- Die Auswahl in der Form der Kategorienliste daneben (settings/KategorieListe). -->
		<div class="lg:w-1/3 flex flex-col gap-1">
			{#if laedt}
				<div class="py-4 text-sm text-center text-on-surface-variant">Lade Vorlagen...</div>
			{:else}
				{#each templates as t, _i (_i)}
					{@const gewaehlt = selectedTemplateId === t.id}
					<button
						type="button"
						aria-current={gewaehlt ? 'true' : undefined}
						class="flex flex-col rounded-2xl px-4 py-3 text-left transition-colors {gewaehlt
							? 'bg-secondary-container text-on-secondary-container'
							: 'text-on-surface-variant hover:bg-surface-container'}"
						onclick={() => {
							selectedTemplateId = t.id;
							errorMessage = '';
						}}
					>
						<span class="truncate text-sm font-medium">{vorlagenName(t.typ)}</span>
						<span class="truncate text-sm {gewaehlt ? 'opacity-80' : 'text-on-surface-variant'}"
							>{t.betreff}</span
						>
					</button>
				{/each}
			{/if}
		</div>

		<div class="lg:w-2/3">
			{#if selectedTemplate}
				<div class="flex flex-col h-full gap-6">
					<Feld
						id="betreff"
						label="Betreff"
						value={selectedTemplate.betreff}
						oninput={updateBetreff}
					/>

					<Feld
						id="text_body"
						label="Text-Inhalt"
						mehrzeilig
						zeilen={12}
						feld="font-mono resize-y"
						value={selectedTemplate.text_body}
						oninput={updateTextBody}
					/>

					<MailVorlagenPlatzhalter typ={selectedTemplate.typ} />

					<div class="flex justify-end pt-2">
						<Button size="lg" onclick={saveTemplate} disabled={isSaving} class="px-6">
							{isSaving ? 'Speichern...' : 'Vorlage Speichern'}
						</Button>
					</div>
				</div>
			{:else if templates.length > 0}
				<div
					class="h-full flex flex-col items-center justify-center p-8 border-2 border-dashed border-outline-variant rounded-2xl text-on-surface-variant"
				>
					<p class="text-sm">Bitte wählen Sie links eine Vorlage aus.</p>
				</div>
			{/if}
		</div>
	</div>
</section>
