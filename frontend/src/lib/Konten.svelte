<!-- @component Konten — die Benutzerkonten: Liste, Anlegen, Bearbeiten, Löschen.

     Was eine Rolle darf, steht im Reiter daneben (BenutzerUndRechte), nicht hier. -->
<script>
	import { TriangleAlert, Plus, X } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import KontenTabelle from './KontenTabelle.svelte';
	import Zugangsanfragen from './Zugangsanfragen.svelte';
	import KontoBearbeitenDialog from './KontoBearbeitenDialog.svelte';
	import KontoLoeschenDialog from './KontoLoeschenDialog.svelte';
	import { apiFetch, leseApiFehler } from './apiFetch.js';
	import {
		leeresBenutzerFormular,
		benutzerFormularAus,
		benutzerNutzlast
	} from './benutzerFormular.js';
	import { snackbars } from './stores/snackbars.svelte.js';
	import Button from './components/ui/Button.svelte';
	import Suchpille from './components/ui/Suchpille.svelte';
	import { fehlertext } from './utils/fehlertext.js';

	/** @type {any[]} */
	let konten = $state.raw([]);
	let laedt = $state(false);
	let suchtext = $state('');

	/** @type {string | null} */
	let fehler = $state(null);

	let formularOffen = $state(false);
	let istAenderung = $state(false);
	// Aufbau, Übernahme und Nutzlast des Formulars: benutzerFormular.js.
	/** @type {any} */
	let formular = $state(leeresBenutzerFormular());
	let speichert = $state(false);

	let loeschfrageOffen = $state(false);
	/** @type {any} */
	let zuLoeschen = $state(null);
	let loescht = $state(false);

	let gefilterteKonten = $derived.by(() => {
		const suche = suchtext.trim().toLowerCase();
		if (!suche) return konten;
		return konten.filter(
			(u) =>
				u.vorname.toLowerCase().includes(suche) ||
				u.nachname.toLowerCase().includes(suche) ||
				u.email.toLowerCase().includes(suche) ||
				(u.barcode_id && u.barcode_id.toLowerCase().includes(suche)) ||
				u.rolle.toLowerCase().includes(suche)
		);
	});

	async function ladeKonten() {
		laedt = true;
		fehler = null;
		try {
			const res = await apiFetch('/api/benutzer');
			if (!res.ok) {
				if (res.status === 403)
					throw new Error('Zugriff verweigert: Nur für System-Administratoren.');
				throw new Error(await leseApiFehler(res));
			}
			konten = await res.json();
		} catch (err) {
			fehler = fehlertext(err);
		} finally {
			laedt = false;
		}
	}

	/** @param {SubmitEvent} e */
	async function speichereKonto(e) {
		e.preventDefault();
		speichert = true;
		fehler = null;
		try {
			const url = istAenderung ? `/api/benutzer/${formular.id}` : '/api/benutzer';
			const method = istAenderung ? 'PUT' : 'POST';
			const payload = benutzerNutzlast(formular);
			const res = await apiFetch(url, {
				method,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});
			if (!res.ok) throw new Error(await leseApiFehler(res));
			formularOffen = false;
			snackbars.addToast(
				istAenderung ? 'Benutzer erfolgreich aktualisiert.' : 'Benutzer erfolgreich angelegt.',
				'success'
			);
			ladeKonten();
		} catch (err) {
			fehler = fehlertext(err);
		} finally {
			speichert = false;
		}
	}

	async function loescheKonto() {
		if (!zuLoeschen) return;
		loescht = true;
		fehler = null;
		try {
			const res = await apiFetch(`/api/benutzer/${zuLoeschen.id}`, { method: 'DELETE' });
			if (!res.ok) {
				// Bei 409 (offene Handapparat-Ausleihen) trägt die Server-Meldung bereits den
				// Hinweis „Bitte zuerst zurückbuchen.“; leseApiFehler packt sie aus dem JSON aus.
				throw new Error(await leseApiFehler(res));
			}
			loeschfrageOffen = false;
			zuLoeschen = null;
			snackbars.addToast('Benutzer erfolgreich gelöscht.', 'success');
			ladeKonten();
		} catch (err) {
			// Der Dialog bleibt offen: Der Satz des Servers sagt, was zu tun ist, und steht
			// deshalb dort, wo die Frage gestellt wurde.
			fehler = fehlertext(err);
		} finally {
			loescht = false;
		}
	}

	function oeffneNeuesKonto() {
		istAenderung = false;
		formular = leeresBenutzerFormular();
		fehler = null;
		formularOffen = true;
	}

	/** @param {any} user */
	function oeffneKonto(user) {
		istAenderung = true;
		formular = benutzerFormularAus(user);
		fehler = null;
		formularOffen = true;
	}

	/** @param {any} user */
	function frageLoeschen(user) {
		zuLoeschen = user;
		fehler = null;
		loeschfrageOffen = true;
	}

	onMount(ladeKonten);
</script>

{#if fehler && !formularOffen && !loeschfrageOffen}
	<div
		role="alert"
		class="flex animate-slide-up items-center gap-2 rounded-xl bg-error-container px-4 py-3 text-sm text-on-error-container"
	>
		<TriangleAlert class="h-4 w-4 shrink-0" aria-hidden="true" />
		<span class="grow">{fehler}</span>
		<button
			type="button"
			class="icon-btn"
			aria-label="Meldung schließen"
			onclick={() => (fehler = null)}
		>
			<X class="h-4 w-4" aria-hidden="true" />
		</button>
	</div>
{/if}

<Zugangsanfragen {konten} onZugeordnet={ladeKonten} />

<!-- `mt-4`: Abstand Reiterband→Pille ist im Haus 24 px + 16 px; er fehlte hier. -->
<div class="mt-4 mb-4 flex flex-col gap-3">
	<Suchpille
		id="benutzer-suchfeld"
		bind:wert={suchtext}
		platzhalter="Name oder E-Mail eingeben …"
		etikett="Benutzer suchen"
	/>
	<div class="flex">
		<Button onclick={oeffneNeuesKonto} class="w-full sm:w-auto">
			<Plus class="h-4 w-4" aria-hidden="true" />
			Benutzer anlegen
		</Button>
	</div>
</div>

<KontenTabelle {laedt} konten={gefilterteKonten} {oeffneKonto} {frageLoeschen} />

<KontoBearbeitenDialog
	open={formularOffen}
	onclose={() => (formularOffen = false)}
	{istAenderung}
	bind:formular
	{speichert}
	{fehler}
	{speichereKonto}
/>

<KontoLoeschenDialog
	open={loeschfrageOffen && !!zuLoeschen}
	onclose={() => {
		loeschfrageOffen = false;
		fehler = null;
	}}
	{zuLoeschen}
	{loescht}
	{fehler}
	{loescheKonto}
/>
