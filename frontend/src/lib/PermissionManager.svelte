<script>
	import { authStore } from './stores/authStore.svelte.js';
	import { apiFetch, apiClient } from './apiFetch.js';
	import { onMount } from 'svelte';
	import UserManagement from './UserManagement.svelte';
	import PermissionsEditor from './PermissionsEditor.svelte';
	import Reiter from './components/ui/Reiter.svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import LadeFehler from './components/ui/LadeFehler.svelte';
	import { toastStore } from './stores/toastStore.svelte.js';
	import { permissionsMetadata } from './permissionMetadata.js';

	// State Runes (Svelte 5)
	let activeSubTab = $state('users'); // "users" | "permissions"

	// Permissions State
	/** @type {Record<string, Record<string, boolean>>} */
	let permissionsState = $state({});
	let loadingPermissions = $state(true);

	// Der Grund, aus dem die Matrix nicht geladen werden konnte.
	/** @type {string | null} */
	let error = $state(null);
	/** @type {Record<string, boolean>} */
	let updatingKeys = $state({});

	// Load permissions
	async function fetchPermissions() {
		loadingPermissions = true;
		error = null;
		try {
			const res = await apiFetch('/api/admin/permissions');
			if (!res.ok) {
				if (res.status === 403)
					throw new Error('Zugriff verweigert: Das Recht „Benutzer & Rechte verwalten" fehlt.');
				throw new Error((await res.text()) || 'Fehler beim Laden der Berechtigungen');
			}
			const data = await res.json();

			/** @type {Record<string, Record<string, boolean>>} */
			// Alle aktiven Rollen vorbelegen, damit ihre Spalte auch ohne Server-Zeile erscheint.
			const newState = { admin: {}, leitung: {}, mitarbeiter: {}, kollegium: {}, helfer: {} };
			data.forEach((/** @type {any} */ item) => {
				if (!newState[item.role]) newState[item.role] = {};
				newState[item.role][item.permission] = item.allowed;
			});
			permissionsState = newState;
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		} finally {
			loadingPermissions = false;
		}
	}

	// Toggle a single permission
	/**
	 * @param {string} role
	 * @param {string} permission
	 * @param {boolean} currentVal
	 */
	async function togglePermission(role, permission, currentVal) {
		if (role === 'admin') return;

		const updateKey = `${role}-${permission}`;
		updatingKeys = { ...updatingKeys, [updateKey]: true };
		const newVal = !currentVal;

		try {
			const res = await apiClient.put('/api/admin/permissions', {
				role,
				permission,
				allowed: newVal
			});

			// Die Begründung des Servers durchreichen statt eines Einheitssatzes: Sie nennt, wer
			// die Matrix ändern darf, und ein 400 die unbekannte Rolle oder das unbekannte Recht.
			if (!res.ok) {
				const grund = await res
					.json()
					.then((/** @type {any} */ d) => d?.error)
					.catch(() => null);
				throw new Error(grund || 'Fehler beim Speichern der Berechtigung.');
			}
			permissionsState[role][permission] = newVal;

			toastStore.addToast('Rechte erfolgreich aktualisiert.', 'success');
		} catch (err) {
			toastStore.addToast(err instanceof Error ? err.message : String(err), 'error');
		} finally {
			const copy = { ...updatingKeys };
			delete copy[updateKey];
			updatingKeys = copy;
		}
	}

	onMount(fetchPermissions);
</script>

<div class="w-full space-y-6 animate-fade-in no-print pb-12">
	<!-- Zwei gleichrangige Aufgaben, zwei M3-Primary-Tabs: Benutzer zuerst, weil das die
	     häufige Aufgabe ist (Kollegin anlegen, Rolle zuweisen); Rollen & Rechte dahinter,
	     weil selten und folgenschwer. -->
	<Reiter
		etikett="Benutzer & Rechte"
		aktiv={activeSubTab}
		onwahl={(id) => (activeSubTab = id)}
		reiter={[
			{ id: 'users', label: 'Benutzer', steuert: 'panel-users' },
			{ id: 'permissions', label: 'Rollen & Rechte', steuert: 'panel-permissions' }
		]}
	/>

	{#if activeSubTab === 'permissions'}
		<div id="panel-permissions" role="tabpanel" aria-labelledby="tab-permissions">
			<!-- Der Satz zur Tragweite gilt dieser Tabelle: Sie steuert Menü und API, und die
			     Drift-Warnung der Betriebsbereitschaft zeigt hierher. -->
			<p class="mb-6 max-w-2xl text-sm text-on-surface-variant">
				Rollen-Rechte. Was hier steht, steuert Menü und API — Abweichungen von der Code-Vorgabe
				meldet die Betriebsbereitschaft.
			</p>
			{#if loadingPermissions}
				<div class="flex justify-center p-12"><Ladekreis size="lg" label="Rechte laden" /></div>
			{:else if error}
				<!-- Ohne geladene Matrix stünden alle Schalter auf „aus", als wären die Rechte entzogen. -->
				<LadeFehler titel="Rechte nicht geladen" text={error} onerneut={fetchPermissions} />
			{:else}
				<PermissionsEditor
					schreibgeschuetzt={authStore.currentUser?.rolle !== 'admin'}
					metadata={permissionsMetadata}
					{permissionsState}
					{updatingKeys}
					onToggle={togglePermission}
				/>
			{/if}
		</div>
	{/if}

	{#if activeSubTab === 'users'}
		<!-- `flex flex-col`: Ohne das kollabiert der `mt-4` der Suchzeile in UserManagement mit
		     dem Außenabstand dieser Tafel, und die Pille sitzt 16 px zu hoch. In einem
		     Flex-Container kollabieren Ränder nicht. -->
		<div id="panel-users" role="tabpanel" aria-labelledby="tab-users" class="flex flex-col">
			<UserManagement />
		</div>
	{/if}
</div>
