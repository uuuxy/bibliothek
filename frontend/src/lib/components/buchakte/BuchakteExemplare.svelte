<script>
	import { melde } from '../../../inventur/lib/store.svelte.js';
	import { loeschenBestaetigen } from '../../stores/bestaetigung.svelte.js';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { hatRecht } from '../../menue.js';
	import { rechteDesReiters } from '../etiketten/druckCenterReiter.js';
	import { druckWarteschlange } from '../../stores/druckWarteschlange.svelte.js';
	import { SvelteSet } from 'svelte/reactivity';
	import { apiFetch, leseApiFehler } from '../../apiFetch.js';
	import { fehlertext } from '../../utils/fehlertext.js';
	import ExemplarKarte from './ExemplarKarte.svelte';
	import { exemplarZahlen } from './exemplarStatus.js';
	import ExemplarEigentumDialog from './ExemplarEigentumDialog.svelte';
	import ExemplarStandortDialog from './ExemplarStandortDialog.svelte';
	import AuswahlLeiste from '../ui/AuswahlLeiste.svelte';
	import Button from '../ui/Button.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { BookOpen, Trash } from '@lucide/svelte';

	/** @type {{ exemplare: any[], titel: any, ladeAlles: (id: string) => void }} */
	let { exemplare = $bindable([]), titel, ladeAlles } = $props();

	const gewaehlteExemplare = new SvelteSet();
	// Exemplare, deren Löschen läuft. Ein zweiter Auftrag vor der Antwort schickte ein zweites
	// DELETE, und nach dem Erfolg des ersten stand „Exemplar nicht gefunden oder schon ausgebucht".
	const loescht = new SvelteSet();
	const markierteLoeschen = $derived([...gewaehlteExemplare].every((id) => loescht.has(id)));
	// Auswahl/Löschen/Barcode/Status hängen an edit_books — nicht an der Rolle.
	const darfBearbeiten = $derived(hatRecht(authStore.currentUser, 'edit_books'));
	// Löschen hängt am Server an delete_books (routen_katalog.go) — die Leiste zeigt den Knopf
	// nur, wer ihn auch benutzen darf (UI entscheidet nach Recht).
	const darfLoeschen = $derived(hatRecht(authStore.currentUser, 'delete_books'));
	// Eigentum markierter Exemplare (4.24, Stufe 3): Dialog wie „Topf der Bestellung ändern".
	let eigentumOffen = $state(false);
	// Standort markierter Exemplare (5.53): derselbe Weg, eigener Dialog.
	let standortOffen = $state(false);
	// Das Etikett entsteht im Druck-Center, im Reiter „Buch-Etiketten" (OFFEN.md 5.5). Wer ihn
	// nicht öffnen darf, bekäme statt des Bogens einen anderen Reiter.
	const darfEtikett = $derived(
		rechteDesReiters('labels').every((recht) => hatRecht(authStore.currentUser, recht))
	);

	/** Dieselbe Übergabe wie Wareneingang und Nachdruck — das Druck-Center öffnet sich selbst. */
	function etikettDrucken(/** @type {any} */ ex) {
		druckWarteschlange.copies = [
			{ barcode_id: ex.barcode_id, titel: titel.title, autor: titel.author }
		];
	}

	/** @param {string} id */
	function toggleSelect(id) {
		if (gewaehlteExemplare.has(id)) {
			gewaehlteExemplare.delete(id);
		} else {
			gewaehlteExemplare.add(id);
		}
	}

	/** Wie Littera „Exemplare: Alle" — ein Klassensatz hat 30 Bände und mehr. */
	function alleAuswaehlen() {
		for (const ex of exemplare) gewaehlteExemplare.add(ex.id);
	}

	/** Nach dem Ändern von Eigentum oder Standort: neu laden (die Werte kommen vom Server) und Markierung weg. */
	async function nachAenderung() {
		gewaehlteExemplare.clear();
		if (titel?.id) await ladeAlles(titel.id);
	}

	/** @param {any} ex */
	async function loescheExemplar(ex) {
		if (loescht.has(ex.id)) return;
		if (!(await loeschenBestaetigen(`Exemplar ${ex.barcode_id} löschen?`))) return;
		loescht.add(ex.id);
		try {
			const res = await apiFetch(`/api/buecher/exemplare/${ex.id}`, {
				method: 'DELETE',
				credentials: 'include'
			});
			if (res.ok) {
				exemplare = exemplare.filter((e) => e.id !== ex.id);
				// Den Kopf aus der Liste nachführen, nach der Grenze der Karten: Ein bestelltes
				// Exemplar stand nie im Bestand, sein Löschen ändert „1 von 2 verfügbar" nicht.
				if (titel) {
					const zahlen = exemplarZahlen(exemplare);
					titel.gesamt = zahlen.bestand;
					if (titel.verfuegbar !== undefined) titel.verfuegbar = zahlen.verfuegbar;
				}
				melde('Exemplar erfolgreich gelöscht', 'success');
			} else {
				melde(await leseApiFehler(res, 'Fehler beim Löschen des Exemplars.'), 'error');
			}
		} catch {
			melde('Netzwerkfehler beim Löschen.', 'error');
		} finally {
			loescht.delete(ex.id);
		}
	}

	async function loescheGewaehlte() {
		const ids = [...gewaehlteExemplare].filter((id) => !loescht.has(id));
		if (ids.length === 0) return;
		if (!(await loeschenBestaetigen(`${ids.length} Exemplare löschen?`))) return;
		for (const id of ids) loescht.add(id);

		const ergebnisse = await Promise.allSettled(
			ids.map(async (id) => {
				const res = await apiFetch(`/api/buecher/exemplare/${id}`, {
					method: 'DELETE',
					credentials: 'include'
				});
				if (!res.ok)
					throw new Error(await leseApiFehler(res, 'Fehler beim Löschen des Exemplars.'));
				return id;
			})
		);
		for (const id of ids) loescht.delete(id);

		// Was nicht gelöscht ist, bleibt markiert; die Meldung nennt den Grund des Servers.
		const gruende = [];
		for (const ergebnis of ergebnisse) {
			if (ergebnis.status === 'rejected') gruende.push(fehlertext(ergebnis.reason));
			else {
				exemplare = exemplare.filter((e) => e.id !== ergebnis.value);
				gewaehlteExemplare.delete(ergebnis.value);
			}
		}
		const geloescht = ergebnisse.length - gruende.length;
		if (geloescht > 0) {
			melde(`${geloescht} Exemplare erfolgreich gelöscht`, 'success');
			if (titel && titel.id) ladeAlles(titel.id);
		}
		if (gruende.length > 0) {
			const anzahl = `${gruende.length} von ${ergebnisse.length} Exemplaren`;
			melde(`${anzahl} nicht gelöscht: ${gruende[0]}`, 'error');
		}
	}
</script>

{#if exemplare.length === 0}
	<div class="py-16 flex flex-col items-center text-on-surface-variant gap-3">
		<BookOpen class="w-10 h-10" aria-hidden="true" />
		<p class="font-semibold text-sm">Keine physischen Exemplare mit Barcodes angelegt.</p>
	</div>
{:else}
	<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
		{#each exemplare as ex (ex.id)}
			<ExemplarKarte
				{darfBearbeiten}
				{darfLoeschen}
				{ex}
				gewaehlt={gewaehlteExemplare.has(ex.id)}
				onToggleSelect={() => toggleSelect(ex.id)}
				loescht={loescht.has(ex.id)}
				onDelete={() => loescheExemplar(ex)}
				onEtikett={darfEtikett ? () => etikettDrucken(ex) : undefined}
			/>
		{/each}
	</div>
	<!-- M3 Selection/Toolbars: die gemeinsame schwebende Leiste unter der Liste, wie auf der
	     Pflegeseite der Schlagworte. -->
	{#if gewaehlteExemplare.size > 0 && darfBearbeiten}
		<AuswahlLeiste
			satz="{gewaehlteExemplare.size} markiert"
			beschriftung="Aktionen für die markierten Exemplare"
			onleeren={() => gewaehlteExemplare.clear()}
		>
			{#if gewaehlteExemplare.size < exemplare.length}
				<Button variant="ghost" onclick={alleAuswaehlen}>Alle auswählen</Button>
			{/if}
			<!-- M3 Toolbars: „Avoid emphasizing more than one action at a time" — eine gefüllte
			     Aktion, die zweite umrandet. -->
			<Button variant="secondary" onclick={() => (standortOffen = true)}>Standort ändern</Button>
			<Button onclick={() => (eigentumOffen = true)}>Eigentum ändern</Button>
			{#if darfLoeschen}
				<Button variant="danger" onclick={loescheGewaehlte} disabled={markierteLoeschen}>
					{#if markierteLoeschen}
						<Ladekreis size="sm" farbe="aktuell" />
					{:else}
						<Trash class="h-4 w-4" aria-hidden="true" />
					{/if}
					Löschen
				</Button>
			{/if}
		</AuswahlLeiste>
	{/if}
{/if}

<ExemplarEigentumDialog
	open={eigentumOffen}
	exemplarIds={Array.from(gewaehlteExemplare)}
	onclose={() => (eigentumOffen = false)}
	onGeaendert={nachAenderung}
/>
<ExemplarStandortDialog
	open={standortOffen}
	exemplarIds={Array.from(gewaehlteExemplare)}
	onclose={() => (standortOffen = false)}
	onGeaendert={nachAenderung}
/>
