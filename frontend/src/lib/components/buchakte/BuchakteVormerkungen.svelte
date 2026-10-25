<script>
	import { apiFetch, apiClient, leseApiFehler } from '../../apiFetch.js';
	import Tabelle from '../ui/Tabelle.svelte';
	import { loeschenBestaetigen } from '../../stores/bestaetigung.svelte.js';
	import { melde } from '../../../inventur/lib/store.svelte.js';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { schuelerRechte } from '../leser/schuelerRechte.js';
	import { sucheSchuelerFuerVormerkung } from './vormerkungSchuelersuche.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import { Clock, Trash } from '@lucide/svelte';

	/** @type {{ vormerkungen: any[], titel: any }} */
	let { vormerkungen = $bindable(), titel } = $props();

	let isAdding = $state(false);
	let suche = $state('');
	let treffer = $state.raw(/** @type {any[]} */ ([]));
	let isSearching = $state(false);
	let notiz = $state('');
	// Die Schülersuche läuft über die Schülerdatei (GET /api/schueler?q=), also gilt deren
	// Recht — dasselbe, das der Server an der Route verlangt (schuelerRechte.js).
	const rechte = $derived(schuelerRechte(authStore.currentUser));

	async function loescheVormerkung(id) {
		if (!(await loeschenBestaetigen('Vormerkung löschen?'))) return;
		try {
			const res = await apiFetch(`/api/vormerkungen/${id}`, { method: 'DELETE' });
			if (res.ok) {
				vormerkungen = vormerkungen.filter((v) => v.id !== id);
				melde('Vormerkung gelöscht', 'success');
			} else {
				melde(await leseApiFehler(res, 'Fehler beim Löschen'), 'error');
			}
		} catch {
			melde('Netzwerkfehler', 'error');
		}
	}

	async function sucheLeser() {
		if (!suche.trim()) {
			treffer = [];
			return;
		}
		isSearching = true;
		try {
			treffer = await sucheSchuelerFuerVormerkung(suche);
		} catch {
			treffer = [];
		} finally {
			isSearching = false;
		}
	}

	async function legeVormerkungAn(studentId) {
		try {
			const res = await apiClient.post('/api/vormerkungen', {
				titel_id: titel.id,
				schueler_id: studentId,
				notiz
			});
			if (res.ok) {
				melde('Erfolgreich vorgemerkt', 'success');
				isAdding = false;
				suche = '';
				treffer = [];
				notiz = '';
				const listenAntwort = await apiFetch(`/api/vormerkungen?titel_id=${titel.id}`);
				// Angelegt ist angelegt — schweigt die Liste, legt jemand sie ein zweites Mal an.
				if (listenAntwort.ok) vormerkungen = await listenAntwort.json();
				else melde('Vormerkung angelegt — die Liste konnte nicht neu geladen werden.', 'error');
			} else {
				melde(await leseApiFehler(res, 'Fehler beim Hinzufügen'), 'error');
			}
		} catch {
			melde('Netzwerkfehler', 'error');
		}
	}
</script>

<!-- Ohne eigene Überschrift: Der Reiter nennt den Inhalt, wie bei den anderen Reitern der Akte. -->
<div class="space-y-6 pt-4">
	<div class="flex justify-end">
		<Button onclick={() => (isAdding = !isAdding)}>
			{isAdding ? 'Abbrechen' : '+ Schüler vormerken'}
		</Button>
	</div>

	{#if isAdding}
		<div class="p-5 bg-surface border border-outline-variant rounded-xl space-y-4 animate-fade-in">
			{#if !rechte.einsehen}
				<!-- Sichtbar statt still: Ein fehlendes Suchfeld sähe wie ein Fehler aus. -->
				<p class="text-sm text-on-surface-variant">
					Die Suche nach Lesern braucht das Recht „Leserdatei anzeigen“ (view_students).
				</p>
			{:else}
				<!-- Flex statt Raster: Die Suche trägt neben sich einen Knopf, die Notiz nicht. -->
				<div class="flex flex-col sm:flex-row items-end gap-4">
					<div class="flex flex-1 w-full items-end gap-2">
						<Feld
							id="student-search-input"
							etikett="Schüler suchen (Name oder Barcode)"
							bind:value={suche}
							onkeydown={(e) => e.key === 'Enter' && sucheLeser()}
							placeholder="z.B. Max Mustermann"
							class="flex-1"
						/>
						<Button variant="secondary" onclick={sucheLeser} disabled={isSearching}>
							{isSearching ? '...' : 'Suchen'}
						</Button>
					</div>
					<Feld
						id="notiz-input"
						etikett="Interne Notiz (optional)"
						bind:value={notiz}
						placeholder="z.B. Braucht es dringend für Referat"
						class="flex-1 w-full"
					/>
				</div>

				{#if treffer.length > 0}
					<div
						class="mt-4 border border-outline-variant rounded-xl overflow-hidden bg-surface-container-lowest"
					>
						{#each treffer as r, _i (_i)}
							<div
								class="flex items-center justify-between p-3 border-b border-outline-variant last:border-0 hover:bg-surface transition-colors"
							>
								<div>
									<p class="font-semibold text-on-surface text-sm">{r.title}</p>
									<p class="text-xs text-on-surface-variant">{r.subtitle}</p>
								</div>
								<Button
									variant="secondary"
									size="sm"
									onclick={() => legeVormerkungAn(r.id)}
									class="border-transparent bg-secondary-container text-on-secondary-container"
								>
									Auswählen
								</Button>
							</div>
						{/each}
					</div>
				{:else if suche && !isSearching && treffer.length === 0}
					<p class="text-sm text-on-surface-variant mt-2">Keine Schüler gefunden.</p>
				{/if}
			{/if}
		</div>
	{/if}

	{#if vormerkungen.length === 0}
		<div
			class="py-12 flex flex-col items-center text-on-surface-variant gap-3 border-2 border-dashed border-outline-variant rounded-xl bg-surface/50"
		>
			<Clock class="w-10 h-10" aria-hidden="true" />
			<p class="font-medium text-sm">Keine ausstehenden Vormerkungen für diesen Titel.</p>
		</div>
	{:else}
		<Tabelle beschriftung="Vormerkungen" class="whitespace-nowrap">
			<thead>
				<tr>
					<th>Wartet seit</th>
					<th>Schüler</th>
					<th>Notiz</th>
					<th class="text-right">Aktion</th>
				</tr>
			</thead>
			<tbody>
				{#each vormerkungen as v, _i (_i)}
					<tr>
						<td class="font-medium">
							{new Date(v.erstellt_am).toLocaleDateString('de-DE', {
								day: '2-digit',
								month: '2-digit',
								year: 'numeric'
							})}
						</td>
						<td class="font-semibold text-primary">
							{v.schueler_name || 'Unbekannt'}
						</td>
						<td>
							{v.notiz || '—'}
						</td>
						<td class="text-right">
							<button
								onclick={() => loescheVormerkung(v.id)}
								class="icon-btn text-error"
								data-tip="Vormerkung löschen"
								aria-label="Vormerkung löschen"
							>
								<Trash class="w-4 h-4" aria-hidden="true" />
							</button>
						</td>
					</tr>
				{/each}
			</tbody>
		</Tabelle>
	{/if}
</div>
