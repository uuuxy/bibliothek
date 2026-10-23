<script>
	import { authStore } from './stores/authStore.svelte.js';
	import { apiFetch, apiClient, leseApiFehler } from './apiFetch.js';
	import { onMount } from 'svelte';
	import Konten from './Konten.svelte';
	import RechteMatrix from './RechteMatrix.svelte';
	import Reiter from './components/ui/Reiter.svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import LadeFehler from './components/ui/LadeFehler.svelte';
	import { snackbars } from './stores/snackbars.svelte.js';
	import { rechteBeschreibungen } from './rechteKategorien.js';
	import { fehlertext } from './utils/fehlertext.js';

	// State Runes (Svelte 5)
	let aktiverReiter = $state('users'); // "users" | "permissions"

	// Permissions State
	/** @type {Record<string, Record<string, boolean>>} */
	let rechteJeRolle = $state({});
	let laedt = $state(true);

	// Der Grund, aus dem die Matrix nicht geladen werden konnte.
	/** @type {string | null} */
	let fehler = $state(null);
	/** @type {Record<string, boolean>} */
	let updatingKeys = $state({});

	// Load permissions
	async function ladeRechte() {
		laedt = true;
		fehler = null;
		try {
			const res = await apiFetch('/api/admin/permissions');
			if (!res.ok) {
				if (res.status === 403)
					throw new Error('Zugriff verweigert: Das Recht „Benutzer & Rechte verwalten" fehlt.');
				throw new Error(await leseApiFehler(res, 'Fehler beim Laden der Berechtigungen'));
			}
			const daten = await res.json();

			/** @type {Record<string, Record<string, boolean>>} */
			// Alle aktiven Rollen vorbelegen, damit ihre Spalte auch ohne Server-Zeile erscheint.
			const newState = { admin: {}, leitung: {}, mitarbeiter: {}, kollegium: {}, helfer: {} };
			daten.forEach((/** @type {any} */ item) => {
				if (!newState[item.role]) newState[item.role] = {};
				newState[item.role][item.permission] = item.allowed;
			});
			rechteJeRolle = newState;
		} catch (err) {
			fehler = fehlertext(err);
		} finally {
			laedt = false;
		}
	}

	// Toggle a single permission
	/**
	 * @param {string} role
	 * @param {string} permission
	 * @param {boolean} currentVal
	 */
	async function schalteRecht(role, permission, currentVal) {
		if (role === 'admin') return;

		const schluessel = `${role}-${permission}`;
		updatingKeys = { ...updatingKeys, [schluessel]: true };
		const newVal = !currentVal;

		try {
			const res = await apiClient.put('/api/admin/permissions', {
				role,
				permission,
				allowed: newVal
			});

			// Die Begründung des Servers durchreichen statt eines Einheitssatzes: Sie nennt, wer
			// die Matrix ändern darf, und ein 400 die unbekannte Rolle oder das unbekannte Recht.
			if (!res.ok)
				throw new Error(await leseApiFehler(res, 'Fehler beim Speichern der Berechtigung.'));
			rechteJeRolle[role][permission] = newVal;

			snackbars.addToast('Rechte erfolgreich aktualisiert.', 'success');
		} catch (err) {
			snackbars.addToast(fehlertext(err), 'error');
		} finally {
			const copy = { ...updatingKeys };
			delete copy[schluessel];
			updatingKeys = copy;
		}
	}

	onMount(ladeRechte);
</script>

<div class="w-full space-y-6 animate-fade-in no-print pb-12">
	<!-- Zwei gleichrangige Aufgaben, zwei M3-Primary-Tabs: Benutzer zuerst, weil das die
	     häufige Aufgabe ist (Kollegin anlegen, Rolle zuweisen); Rollen & Rechte dahinter,
	     weil selten und folgenschwer. -->
	<Reiter
		etikett="Benutzer & Rechte"
		aktiv={aktiverReiter}
		onwahl={(id) => (aktiverReiter = id)}
		reiter={[
			{ id: 'users', label: 'Benutzer', steuert: 'panel-users' },
			{ id: 'permissions', label: 'Rollen & Rechte', steuert: 'panel-permissions' }
		]}
	/>

	{#if aktiverReiter === 'permissions'}
		<div id="panel-permissions" role="tabpanel" aria-labelledby="tab-permissions">
			<!-- Der Satz zur Tragweite gilt dieser Tabelle: Sie steuert Menü und API, und die
			     Drift-Warnung der Betriebsbereitschaft zeigt hierher. -->
			<p class="mb-6 max-w-2xl text-sm text-on-surface-variant">
				Rollen-Rechte. Was hier steht, steuert Menü und API — Abweichungen von der Code-Vorgabe
				meldet die Betriebsbereitschaft.
			</p>
			{#if laedt}
				<div class="flex justify-center p-12"><Ladekreis size="lg" etikett="Rechte laden" /></div>
			{:else if fehler}
				<!-- Ohne geladene Matrix stünden alle Schalter auf „aus", als wären die Rechte entzogen. -->
				<LadeFehler titel="Rechte nicht geladen" text={fehler} onerneut={ladeRechte} />
			{:else}
				<RechteMatrix
					schreibgeschuetzt={authStore.currentUser?.rolle !== 'admin'}
					beschreibungen={rechteBeschreibungen}
					{rechteJeRolle}
					{updatingKeys}
					onToggle={schalteRecht}
				/>
			{/if}
		</div>
	{/if}

	{#if aktiverReiter === 'users'}
		<!-- `flex flex-col`: Ohne das kollabiert der `mt-4` der Suchzeile in Konten mit
		     dem Außenabstand dieser Tafel, und die Pille sitzt 16 px zu hoch. In einem
		     Flex-Container kollabieren Ränder nicht. -->
		<div id="panel-users" role="tabpanel" aria-labelledby="tab-users" class="flex flex-col">
			<Konten />
		</div>
	{/if}
</div>
