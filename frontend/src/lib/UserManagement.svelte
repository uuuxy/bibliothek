<!-- @component UserManagement — die Benutzerkonten: Liste, Anlegen, Bearbeiten, Löschen.

     Was eine Rolle darf, steht im Reiter daneben (PermissionManager), nicht hier. -->
<script>
	import { TriangleAlert, Plus, X } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import UserManagementTable from './UserManagementTable.svelte';
	import UserManagementZugangsanfragen from './UserManagementZugangsanfragen.svelte';
	import UserManagementEditModal from './UserManagementEditModal.svelte';
	import UserManagementDeleteModal from './UserManagementDeleteModal.svelte';
	import { apiFetch, extractApiError } from './apiFetch.js';
	import {
		leeresBenutzerFormular,
		benutzerFormularAus,
		benutzerNutzlast
	} from './benutzerFormular.js';
	import { toastStore } from './stores/toastStore.svelte.js';
	import Button from './components/ui/Button.svelte';
	import Suchpille from './components/ui/Suchpille.svelte';
	import { fehlertext } from './utils/fehlertext.js';

	/** @type {any[]} */
	let users = $state.raw([]);
	let loadingUsers = $state(false);
	let userSearchQuery = $state('');

	/** @type {string | null} */
	let error = $state(null);

	let showUserModal = $state(false);
	let isEditingUser = $state(false);
	// Aufbau, Übernahme und Nutzlast des Formulars: benutzerFormular.js.
	/** @type {any} */
	let userForm = $state(leeresBenutzerFormular());
	let submittingUser = $state(false);

	let showDeleteConfirm = $state(false);
	/** @type {any} */
	let userToDelete = $state(null);
	let deletingUser = $state(false);

	let filteredUsers = $derived.by(() => {
		const query = userSearchQuery.trim().toLowerCase();
		if (!query) return users;
		return users.filter(
			(u) =>
				u.vorname.toLowerCase().includes(query) ||
				u.nachname.toLowerCase().includes(query) ||
				u.email.toLowerCase().includes(query) ||
				(u.barcode_id && u.barcode_id.toLowerCase().includes(query)) ||
				u.rolle.toLowerCase().includes(query)
		);
	});

	async function fetchUsers() {
		loadingUsers = true;
		error = null;
		try {
			const res = await apiFetch('/api/benutzer');
			if (!res.ok) {
				if (res.status === 403)
					throw new Error('Zugriff verweigert: Nur für System-Administratoren.');
				throw new Error(await extractApiError(res));
			}
			users = await res.json();
		} catch (err) {
			error = fehlertext(err);
		} finally {
			loadingUsers = false;
		}
	}

	/** @param {SubmitEvent} e */
	async function handleSaveUser(e) {
		e.preventDefault();
		submittingUser = true;
		error = null;
		try {
			const url = isEditingUser ? `/api/benutzer/${userForm.id}` : '/api/benutzer';
			const method = isEditingUser ? 'PUT' : 'POST';
			const payload = benutzerNutzlast(userForm);
			const res = await apiFetch(url, {
				method,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});
			if (!res.ok) throw new Error(await extractApiError(res));
			showUserModal = false;
			toastStore.addToast(
				isEditingUser ? 'Benutzer erfolgreich aktualisiert.' : 'Benutzer erfolgreich angelegt.',
				'success'
			);
			fetchUsers();
		} catch (err) {
			error = fehlertext(err);
		} finally {
			submittingUser = false;
		}
	}

	async function confirmDeleteUser() {
		if (!userToDelete) return;
		deletingUser = true;
		error = null;
		try {
			const res = await apiFetch(`/api/benutzer/${userToDelete.id}`, { method: 'DELETE' });
			if (!res.ok) {
				// Bei 409 (offene Handapparat-Ausleihen) trägt die Server-Meldung bereits den
				// Hinweis "bitte zuerst zurückbuchen"; extractApiError packt sie aus dem JSON aus.
				throw new Error(await extractApiError(res));
			}
			showDeleteConfirm = false;
			userToDelete = null;
			toastStore.addToast('Benutzer erfolgreich gelöscht.', 'success');
			fetchUsers();
		} catch (err) {
			// Der Dialog bleibt offen: Der Satz des Servers sagt, was zu tun ist, und steht
			// deshalb dort, wo die Frage gestellt wurde.
			error = fehlertext(err);
		} finally {
			deletingUser = false;
		}
	}

	function openNewUserModal() {
		isEditingUser = false;
		userForm = leeresBenutzerFormular();
		error = null;
		showUserModal = true;
	}

	/** @param {any} user */
	function openEditUserModal(user) {
		isEditingUser = true;
		userForm = benutzerFormularAus(user);
		error = null;
		showUserModal = true;
	}

	/** @param {any} user */
	function openDeleteConfirm(user) {
		userToDelete = user;
		error = null;
		showDeleteConfirm = true;
	}

	onMount(fetchUsers);
</script>

{#if error && !showUserModal && !showDeleteConfirm}
	<div
		role="alert"
		class="flex animate-slide-up items-center gap-2 rounded-xl bg-error-container px-4 py-3 text-sm text-on-error-container"
	>
		<TriangleAlert class="h-4 w-4 shrink-0" aria-hidden="true" />
		<span class="grow">{error}</span>
		<button
			type="button"
			class="icon-btn"
			aria-label="Meldung schließen"
			onclick={() => (error = null)}
		>
			<X class="h-4 w-4" aria-hidden="true" />
		</button>
	</div>
{/if}

<UserManagementZugangsanfragen {users} onZugeordnet={fetchUsers} />

<!-- `mt-4`: Abstand Reiterband→Pille ist im Haus 24 px + 16 px; er fehlte hier. -->
<div class="mt-4 mb-4 flex flex-col gap-3">
	<Suchpille
		id="benutzer-suchfeld"
		bind:wert={userSearchQuery}
		platzhalter="Name oder E-Mail eingeben …"
		etikett="Benutzer suchen"
	/>
	<div class="flex">
		<Button onclick={openNewUserModal} class="w-full sm:w-auto">
			<Plus class="h-4 w-4" aria-hidden="true" />
			Benutzer anlegen
		</Button>
	</div>
</div>

<UserManagementTable {loadingUsers} {filteredUsers} {openEditUserModal} {openDeleteConfirm} />

<UserManagementEditModal
	open={showUserModal}
	onclose={() => (showUserModal = false)}
	{isEditingUser}
	bind:userForm
	{submittingUser}
	{error}
	{handleSaveUser}
/>

<UserManagementDeleteModal
	open={showDeleteConfirm && !!userToDelete}
	onclose={() => {
		showDeleteConfirm = false;
		error = null;
	}}
	{userToDelete}
	{deletingUser}
	{error}
	{confirmDeleteUser}
/>
