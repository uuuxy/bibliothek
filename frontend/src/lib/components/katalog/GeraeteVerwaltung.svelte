<script>
	import { onMount } from 'svelte';
	import { apiClient, leseApiFehler } from '../../apiFetch.js';
	import { erzeugeGeraeteListe } from './geraeteListe.svelte.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import { leeresGeraetFormular, geraetFormularAus, geraetNutzlast } from './geraetFormular.js';
	import Button from '../ui/Button.svelte';
	import GeraetFelder from './GeraetFelder.svelte';
	import { anzahlText } from '../../utils/format.js';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import SuchZustand from '../ui/SuchZustand.svelte';
	import Icon from '../ui/Icon.svelte';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { hatRecht } from '../../menue.js';

	// Anlegen, Ändern und „Defekt melden" hängen am Server an edit_books (routen_katalog.go), die
	// Liste an view_books. Ohne das Recht bleibt die Liste lesbar und verliert nur die Aktionen, wie
	// KlassenKarte und Signaturen: niemandem einen Knopf anbieten, der im 403 endet.
	const darfBearbeiten = $derived(hatRecht(authStore.currentUser, 'edit_books'));

	/**
	 * Geräte-Verwaltung, ein Bereich im Medienkatalog und kein eigener Menüpunkt: Ausgebucht
	 * wird ohnehin über die Omnibox mit G--Barcode.
	 * Anlegen, Liste mit aktuellem Ausleiher, Defekt-Schalter, Stammdaten-Pflege.
	 */

	// Laden, Ladezustand und Ladefehler liegen in geraeteListe.svelte.js (Größen-Ratsche).
	const geraete = erzeugeGeraeteListe();
	let formularOffen = $state(false);
	/** @type {string | null} */
	let bearbeiteId = $state(null);
	/** @type {any} */
	let formular = $state(leeresGeraetFormular());
	let speichert = $state(false);

	onMount(geraete.lade);

	function oeffneAnlegen() {
		bearbeiteId = null;
		formular = leeresGeraetFormular();
		formularOffen = true;
	}

	/** @param {any} g */
	function oeffneBearbeiten(g) {
		bearbeiteId = g.id;
		formular = geraetFormularAus(g);
		formularOffen = true;
	}

	function schliesseFormular() {
		formularOffen = false;
		bearbeiteId = null;
	}

	async function speichere() {
		// Nach „Abbrechen" kann schon ein anderes Formular offen stehen; die Antwort schließt nur ihres.
		const [id, gesendet] = [bearbeiteId, formular];
		speichert = true;
		try {
			// Ein vorhandenes Gerät schickt nur das Geänderte (geraetFormular.js).
			const res = id
				? await apiClient.put(`/api/geraete/${id}`, geraetNutzlast(gesendet, true))
				: await apiClient.post('/api/geraete', geraetNutzlast(gesendet, false));
			if (res.ok) {
				snackbars.addToast(id ? 'Gerät gespeichert' : 'Gerät angelegt', 'success');
				if (formular === gesendet) schliesseFormular();
				geraete.lade();
			} else {
				snackbars.addToast(await leseApiFehler(res, 'Speichern fehlgeschlagen'), 'error');
			}
		} catch {
			snackbars.addToast('Netzwerkfehler', 'error');
		} finally {
			speichert = false;
		}
	}

	/** @param {any} g */
	async function schalteDefekt(g) {
		try {
			// Nur das Kennzeichen: Die Zeile der Liste kann älter sein als das Gerät.
			const res = await apiClient.put(`/api/geraete/${g.id}`, {
				ist_ausleihbar: !g.ist_ausleihbar
			});
			if (res.ok) {
				geraete.lade();
			} else {
				snackbars.addToast(await leseApiFehler(res, 'Statuswechsel fehlgeschlagen'), 'error');
			}
		} catch {
			snackbars.addToast('Netzwerkfehler', 'error');
		}
	}
</script>

<div class="space-y-6 pt-2">
	<!-- Anlegen vorn bei der Liste wie in Leserdatei und Benutzern, nicht am fernen Rand. -->
	<div class="space-y-3 border-b border-outline-variant pb-3">
		<div>
			<h2 class="text-title-medium text-on-surface">
				{anzahlText(geraete.liste.length, 'Gerät', 'Geräte')}
			</h2>
			<p class="mt-0.5 text-sm text-on-surface-variant">
				Ausgebucht wird am Kiosk über den G--Barcode; das Zubehör wird dort als Checkliste bestätigt
			</p>
		</div>
		{#if darfBearbeiten}
			<Button variant="primary" onclick={oeffneAnlegen}>
				<Icon name="add" />
				Gerät anlegen
			</Button>
		{/if}
	</div>

	{#if formularOffen}
		<div class="rounded-xl border border-outline-variant p-4">
			<GeraetFelder bind:formular barcodeFest={bearbeiteId != null} />
			<div class="mt-4 flex justify-end gap-2">
				<Button variant="secondary" onclick={schliesseFormular}>Abbrechen</Button>
				<Button variant="primary" onclick={speichere} disabled={speichert}>
					{bearbeiteId ? 'Speichern' : 'Anlegen'}
				</Button>
			</div>
		</div>
	{/if}

	<!-- Laden, Fehler und leere Liste wie auf den übrigen Verwaltungsseiten (AnliegenListe,
	     BestellHistorie). -->
	{#if geraete.laedt}
		<div class="flex justify-center py-16"><Ladekreis size="lg" verzoegert /></div>
	{:else if geraete.ladefehler}
		<LadeFehler onerneut={geraete.lade} titel="Geräte nicht geladen" text={geraete.ladefehler} />
	{:else if geraete.liste.length === 0}
		<SuchZustand symbol="devices" titel="Noch keine Geräte erfasst" />
	{:else}
		<ul class="divide-y divide-outline-variant">
			{#each geraete.liste as g (g.id)}
				<!-- Ohne view_students schickt der Server den Namen als leeren Text: verliehen bleibt es. -->
				{@const verliehen = g.ausgeliehen_an != null}
				<li class="flex items-center justify-between gap-4 py-4">
					<div class="min-w-0 flex-1">
						<p class="truncate text-body-medium-emphasized text-on-surface">{g.modellname}</p>
						<p class="mt-0.5 text-xs text-on-surface-variant">
							{g.barcode_id}{#if g.seriennummer}&nbsp;· SN {g.seriennummer}{/if}
							{#if g.zubehoer}&nbsp;· Zubehör: {g.zubehoer}{/if}
						</p>
						{#if g.zustand_notiz}
							<p class="mt-0.5 text-xs italic text-on-surface-variant">{g.zustand_notiz}</p>
						{/if}
					</div>
					<span
						class="shrink-0 rounded-full px-2 py-0.5 text-label-small {verliehen
							? 'bg-secondary-container text-on-secondary-container'
							: g.ist_ausleihbar
								? 'bg-primary-container text-on-primary-container'
								: 'bg-error-container text-on-error-container'}"
					>
						{verliehen
							? g.ausgeliehen_an
								? `verliehen an ${g.ausgeliehen_an}`
								: 'verliehen'
							: g.ist_ausleihbar
								? 'im Schrank'
								: 'defekt oder gesperrt'}
					</span>
					<!-- Bearbeiten als Symbolknopf wie in jeder Zeile; „Defekt melden" schaltet einen Zustand
					     um, den ein Symbol allein nicht sagt. 16 px Abstand für die Trefferfläche. -->
					{#if darfBearbeiten}
						<div class="flex shrink-0 items-center gap-4">
							<Button
								variant="symbol"
								size="sm"
								onclick={() => oeffneBearbeiten(g)}
								data-tip="Bearbeiten"
								aria-label="{g.modellname} bearbeiten"
							>
								<Icon name="edit" />
							</Button>
							<Button
								variant={g.ist_ausleihbar ? 'danger' : 'secondary'}
								size="sm"
								onclick={() => schalteDefekt(g)}
							>
								<Icon name="build" groesse={18} />
								{g.ist_ausleihbar ? 'Defekt melden' : 'Wieder freigeben'}
							</Button>
						</div>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</div>
