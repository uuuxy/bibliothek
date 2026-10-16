<script>
	import { melde } from '../inventur/lib/store.svelte.js';
	import { loeschenBestaetigen } from './stores/bestaetigung.svelte.js';
	import { authStore } from './stores/authStore.svelte.js';
	import { erlaubteBildschirme, hatRecht } from './menue.js';
	import { druckWarteschlange } from './stores/druckWarteschlange.svelte.js';
	import { SvelteSet } from 'svelte/reactivity';
	import { apiFetch } from './apiFetch.js';
	import ExemplarKarte from './components/ExemplarKarte.svelte';
	import { exemplarZahlen } from './components/exemplarStatus.js';
	import ExemplarEigentumDialog from './components/ExemplarEigentumDialog.svelte';
	import ExemplarStandortDialog from './components/ExemplarStandortDialog.svelte';
	import AuswahlLeiste from './components/ui/AuswahlLeiste.svelte';
	import Button from './components/ui/Button.svelte';
	import { BookOpen, Trash } from '@lucide/svelte';

	/** @type {{ exemplare: any[], titel: any, ladeAlles: (id: string) => void }} */
	let { exemplare = $bindable([]), titel, ladeAlles } = $props();

	const gewaehlteExemplare = new SvelteSet();
	// Auswahl/Löschen/Barcode/Status hängen an edit_books — nicht an der Rolle.
	const darfBearbeiten = $derived(hatRecht(authStore.currentUser, 'edit_books'));
	// Löschen hängt am Server an delete_books (routen_katalog.go) — die Leiste zeigt den Knopf
	// nur, wer ihn auch benutzen darf (UI entscheidet nach Recht).
	const darfLoeschen = $derived(hatRecht(authStore.currentUser, 'delete_books'));
	// Eigentum markierter Exemplare (4.24, Stufe 3): Dialog wie „Topf der Bestellung ändern".
	let eigentumOffen = $state(false);
	// Standort markierter Exemplare (5.53): derselbe Weg, eigener Dialog.
	let standortOffen = $state(false);
	// Das Etikett entsteht im Druck-Center, auf dem Bogen nach der Vorlage (OFFEN.md 5.5). Wer
	// den Bildschirm nicht öffnen darf, bekäme statt des Bogens den ersten erlaubten: Der
	// Router stellt einen gesperrten Reiter zurück.
	const darfEtikett = $derived(
		darfBearbeiten && erlaubteBildschirme(authStore.currentUser).has('druck-center')
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
		if (!(await loeschenBestaetigen(`Exemplar ${ex.barcode_id} löschen?`))) return;
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
				const err = await res.json().catch(() => ({}));
				melde(err.error || 'Fehler beim Löschen des Exemplars.', 'error');
			}
		} catch {
			melde('Netzwerkfehler beim Löschen.', 'error');
		}
	}

	async function loescheGewaehlte() {
		if (gewaehlteExemplare.size === 0) return;
		if (!(await loeschenBestaetigen(`${gewaehlteExemplare.size} Exemplare löschen?`))) return;

		let successCount = 0;

		const ergebnisse = await Promise.allSettled(
			Array.from(gewaehlteExemplare).map(async (id) => {
				const res = await apiFetch(`/api/buecher/exemplare/${id}`, {
					method: 'DELETE',
					credentials: 'include'
				});
				if (!res.ok) throw new Error('not ok');
				return id;
			})
		);

		for (const ergebnis of ergebnisse) {
			if (ergebnis.status === 'fulfilled') {
				const id = ergebnis.value;
				exemplare = exemplare.filter((e) => e.id !== id);
				successCount++;
			} else {
				console.error('Fehler beim Löschen:', ergebnis.reason);
			}
		}
		gewaehlteExemplare.clear();
		if (successCount > 0) {
			melde(`${successCount} Exemplare erfolgreich gelöscht`, 'success');
			if (titel && titel.id) ladeAlles(titel.id);
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
				{ex}
				gewaehlt={gewaehlteExemplare.has(ex.id)}
				onToggleSelect={() => toggleSelect(ex.id)}
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
				<Button variant="danger" onclick={loescheGewaehlte}>
					<Trash class="h-4 w-4" aria-hidden="true" />
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
