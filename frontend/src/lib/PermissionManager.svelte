<script>
	import { authStore } from './stores/authStore.svelte.js';
	import { AlertTriangle, Check } from '@lucide/svelte';
	import { apiFetch, apiClient } from './apiFetch.js';
	import { onMount } from 'svelte';
	import UserManagement from './UserManagement.svelte';
	import PermissionsEditor from './PermissionsEditor.svelte';
	import Reiter from './components/ui/Reiter.svelte';
	import { permissionsMetadata } from './permissionMetadata.js';

	// State Runes (Svelte 5)
	let activeSubTab = $state('users'); // "users" | "permissions"

	// Permissions State
	/** @type {Record<string, Record<string, boolean>>} */
	let permissionsState = $state({});
	let loadingPermissions = $state(true);

	// Common UI State
	/** @type {string | null} */
	let error = $state(null);
	/** @type {Record<string, boolean>} */
	let updatingKeys = $state({});
	/** @type {string | null} */
	let successMessage = $state(null);

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
			// Alle vier aktiven Rollen vorbelegen, damit ihre Spalte auch ohne Server-Zeile
			// erscheint. 'kollegium' statt des seit Migration 069 toten 'lehrer' — der alte
			// Key wurde von keinem Consumer gelesen (PermissionsEditor liest 'kollegium'),
			// kollegium selbst entstand bisher nur zufällig über die Rückfallzeile unten.
			const newState = { admin: {}, mitarbeiter: {}, kollegium: {}, helfer: {} };
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

			// Die Begründung des Servers durchreichen statt sie durch einen Einheitssatz zu
			// ersetzen. Seit die Rechte-Matrix Administratoren vorbehalten ist, ist genau
			// diese Begründung die Information, die zählt ("nur ein Administrator"), und
			// ein 400 nennt die unbekannte Rolle/Rechte-Kombination beim Namen.
			if (!res.ok) {
				const grund = await res
					.json()
					.then((/** @type {any} */ d) => d?.error)
					.catch(() => null);
				throw new Error(grund || 'Fehler beim Speichern der Berechtigung.');
			}
			permissionsState[role][permission] = newVal;

			showToast('Rechte erfolgreich aktualisiert.');
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
			setTimeout(() => {
				error = null;
			}, 5000);
		} finally {
			const copy = { ...updatingKeys };
			delete copy[updateKey];
			updatingKeys = copy;
		}
	}

	// Helper: Flash message toast
	/** @param {string} msg */
	function showToast(msg) {
		successMessage = msg;
		setTimeout(() => {
			if (successMessage === msg) successMessage = null;
		}, 3000);
	}

	onMount(fetchPermissions);
</script>

<div class="w-full space-y-6 animate-fade-in no-print pb-12">
	<!-- Zwei gleichrangige Aufgaben, zwei M3-Primary-Tabs links oben (Peters Entscheidung
	     26.08.2026): Benutzer zuerst, weil das die häufige Aufgabe ist (Kollegin anlegen,
	     Rolle zuweisen); Rollen & Rechte dahinter, weil selten und folgenschwer. Vorher: ein
	     Segmented-Button mit Emoji rechts in der Ecke, 12 px, unter einem Menüpunkt, der
	     nur „Berechtigungen" hieß — wer ein Konto suchte, klickte in die Einstellungen. -->
	<Reiter
		etikett="Benutzer & Rechte"
		aktiv={activeSubTab}
		onwahl={(id) => (activeSubTab = id)}
		reiter={[
			{ id: 'users', label: 'Benutzer', steuert: 'panel-users' },
			{ id: 'permissions', label: 'Rollen & Rechte', steuert: 'panel-permissions' }
		]}
	/>

	<!-- Error Alerts -->
	{#if error}
		<div
			class="p-4 rounded-2xl bg-rose-50 border border-rose-100 text-rose-600 text-sm font-medium transition-all animate-slide-up flex items-center justify-between"
		>
			<span><AlertTriangle class="h-4 w-4" aria-hidden="true" /> {error}</span>
			<button
				onclick={() => (error = null)}
				class="text-rose-500 hover:text-rose-600 font-bold ml-2">×</button
			>
		</div>
	{/if}

	<!-- Success Toast -->
	{#if successMessage}
		<div
			class="fixed bottom-6 right-6 z-50 p-4 rounded-xl bg-emerald-50 border border-emerald-100 text-emerald-700 text-xs font-semibold shadow-lg transition-all animate-slide-up flex items-center gap-2"
		>
			<Check class="h-4 w-4 text-emerald-600" aria-hidden="true" />
			<span>{successMessage}</span>
		</div>
	{/if}

	{#if activeSubTab === 'permissions'}
		<div id="panel-permissions" role="tabpanel" aria-labelledby="tab-permissions">
			{#if loadingPermissions}
				<div class="p-12 text-center text-slate-400 font-medium animate-pulse">
					Lade Rechtekonfiguration...
				</div>
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
		<div id="panel-users" role="tabpanel" aria-labelledby="tab-users"><UserManagement /></div>
	{/if}
</div>
