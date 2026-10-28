<!-- @component BuchakteVormerken — „+ Schüler vormerken" im Reiter Vormerkungen: Schüler suchen,
     Notiz, Auswählen. Die Liste darunter führt BuchakteVormerkungen. -->
<script>
	import { apiFetch, apiClient, leseApiFehler } from '../../apiFetch.js';
	import { melde } from '../../../inventur/lib/store.svelte.js';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { schuelerRechte } from '../leser/schuelerRechte.js';
	import { sucheSchuelerFuerVormerkung } from './vormerkungSchuelersuche.js';
	import { fehlertext } from '../../utils/fehlertext.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';

	/** @type {{ vormerkungen: any[], titel: any }} */
	let { vormerkungen = $bindable(), titel } = $props();

	let isAdding = $state(false);
	let suche = $state('');
	let treffer = $state.raw(/** @type {any[]} */ ([]));
	let isSearching = $state(false);
	let suchFehler = $state('');
	let notiz = $state('');
	// Die Schülersuche läuft über die Schülerdatei (GET /api/schueler?q=), also gilt deren
	// Recht — dasselbe, das der Server an der Route verlangt (schuelerRechte.js).
	const rechte = $derived(schuelerRechte(authStore.currentUser));
	// Der Schüler, dessen Vormerkung gerade angelegt wird. Ein zweiter Klick vor der Antwort
	// schickte eine zweite Anfrage, und nach „Erfolgreich vorgemerkt" stand „bereits vorgemerkt".
	/** @type {string | null} */
	let legtAn = $state(null);

	async function sucheLeser() {
		treffer = [];
		suchFehler = '';
		if (!suche.trim()) return;
		isSearching = true;
		try {
			treffer = await sucheSchuelerFuerVormerkung(suche);
		} catch (e) {
			suchFehler = fehlertext(e);
		} finally {
			isSearching = false;
		}
	}

	const NACHLADEN_GESCHEITERT = 'Vormerkung angelegt — die Liste konnte nicht neu geladen werden';
	/** @param {string} studentId */
	async function legeVormerkungAn(studentId) {
		if (legtAn) return;
		legtAn = studentId;
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
				else melde(`${NACHLADEN_GESCHEITERT}: ${await leseApiFehler(listenAntwort)}`, 'error');
			} else {
				melde(await leseApiFehler(res, 'Fehler beim Hinzufügen'), 'error');
			}
		} catch {
			melde('Netzwerkfehler', 'error');
		} finally {
			legtAn = null;
		}
	}
</script>

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
							<!-- Gesperrt, solange eine Vormerkung angelegt wird; der Ladekreis steht nur im
							     gewählten Knopf (M3, Progress indicators). -->
							<Button
								variant="secondary"
								size="sm"
								onclick={() => legeVormerkungAn(r.id)}
								disabled={legtAn !== null}
								class="border-transparent bg-secondary-container text-on-secondary-container"
							>
								{#if legtAn === r.id}
									<Ladekreis size="sm" farbe="aktuell" />
								{/if}
								Auswählen
							</Button>
						</div>
					{/each}
				</div>
			{:else if suchFehler}
				<p role="alert" class="text-sm text-error mt-2">{suchFehler}</p>
			{:else if suche && !isSearching && treffer.length === 0}
				<p class="text-sm text-on-surface-variant mt-2">Keine Schüler gefunden.</p>
			{/if}
		{/if}
	</div>
{/if}
