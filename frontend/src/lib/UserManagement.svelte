<script>
	/**
	 * UserManagement — self-contained component for staff user CRUD.
	 *
	 * Responsibilities: listing, creating, editing, and deleting system users
	 * (benutzer/benutzer_rollen tables). Isolated from role-permission management.
	 *
	 * State:
	 *   $state users         — loaded from /api/benutzer
	 *   $state userForm      — controlled form for create/edit modal
	 *   $state showUserModal — controls the create/edit dialog
	 *   $state showDeleteConfirm — controls the delete confirmation dialog
	 *   $derived filteredUsers — search-filtered view of users
	 */
	import { onMount } from 'svelte';
	import UserManagementTable from './UserManagementTable.svelte';
	import UserManagementEditModal from './UserManagementEditModal.svelte';
	import UserManagementDeleteModal from './UserManagementDeleteModal.svelte';
	import { apiFetch, extractApiError } from './apiFetch.js';

	/** @type {any[]} */
	let users = $state.raw([]);
	let loadingUsers = $state(false);
	let userSearchQuery = $state('');

	/** @type {string | null} */
	let error = $state(null);
	/** @type {string | null} */
	let successMessage = $state(null);

	// Create / Edit modal state
	let showUserModal = $state(false);
	let isEditingUser = $state(false);
	/** @type {any} */
	let userForm = $state({
		id: '',
		barcode_id: '',
		vorname: '',
		nachname: '',
		email: '',
		rolle: 'mitarbeiter',
		aktiv: true,
		password: ''
	});
	let submittingUser = $state(false);

	// Delete confirmation state
	let showDeleteConfirm = $state(false);
	/** @type {any} */
	let userToDelete = $state(null);
	let deletingUser = $state(false);

	// Search-filtered view — recomputed reactively whenever users or query changes
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
			error = err instanceof Error ? err.message : String(err);
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
			const payload = {
				barcode_id: userForm.barcode_id,
				vorname: userForm.vorname,
				nachname: userForm.nachname,
				email: userForm.email,
				rolle: userForm.rolle,
				aktiv: userForm.aktiv,
				password: userForm.password
			};
			const res = await apiFetch(url, {
				method,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});
			if (!res.ok) {
				throw new Error(await extractApiError(res));
			}
			showUserModal = false;
			showToast(
				isEditingUser ? 'Benutzer erfolgreich aktualisiert.' : 'Benutzer erfolgreich angelegt.'
			);
			fetchUsers();
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
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
			showToast('Benutzer erfolgreich gelöscht.');
			fetchUsers();
		} catch (err) {
			// Modal bewusst offen lassen: Die (handlungsleitende) Fehlermeldung wird inline
			// im Lösch-Dialog angezeigt, nicht im hier ausgeblendeten globalen Banner.
			error = err instanceof Error ? err.message : String(err);
		} finally {
			deletingUser = false;
		}
	}

	function openNewUserModal() {
		isEditingUser = false;
		userForm = {
			id: '',
			barcode_id: '',
			vorname: '',
			nachname: '',
			email: '',
			rolle: 'mitarbeiter',
			aktiv: true,
			password: ''
		};
		error = null;
		showUserModal = true;
	}

	/** @param {any} user */
	function openEditUserModal(user) {
		isEditingUser = true;
		userForm = {
			id: user.id,
			barcode_id: user.barcode_id || '',
			vorname: user.vorname,
			nachname: user.nachname,
			email: user.email,
			rolle: user.rolle,
			aktiv: user.aktiv,
			password: ''
		};
		error = null;
		showUserModal = true;
	}

	/** @param {any} user */
	function openDeleteConfirm(user) {
		userToDelete = user;
		error = null;
		showDeleteConfirm = true;
	}

	/** @param {string} msg */
	function showToast(msg) {
		successMessage = msg;
		setTimeout(() => {
			if (successMessage === msg) successMessage = null;
		}, 3000);
	}

	onMount(fetchUsers);
</script>

<!-- Error Banner -->
{#if error && !showUserModal && !showDeleteConfirm}
	<div
		class="p-4 rounded-2xl bg-rose-50 border border-rose-100 text-rose-600 text-sm font-medium animate-slide-up flex items-center justify-between"
	>
		<span>⚠️ {error}</span>
		<button onclick={() => (error = null)} class="text-rose-450 hover:text-rose-650 font-bold ml-2"
			>×</button
		>
	</div>
{/if}

<!-- Success Toast -->
{#if successMessage}
	<div
		class="fixed bottom-6 right-6 z-50 p-4 rounded-xl bg-emerald-50 border border-emerald-100 text-emerald-700 text-xs font-semibold shadow-lg animate-slide-up flex items-center gap-2"
	>
		<svg
			xmlns="http://www.w3.org/2000/svg"
			class="h-4 w-4 text-emerald-600"
			fill="none"
			viewBox="0 0 24 24"
			stroke="currentColor"
			stroke-width="2.5"
		>
			<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
		</svg>
		<span>{successMessage}</span>
	</div>
{/if}

<!-- Toolbar -->
<div class="flex flex-col sm:flex-row items-center justify-between gap-4 mb-4">
	<div class="relative w-full sm:max-w-xs">
		<input
			type="text"
			bind:value={userSearchQuery}
			placeholder="Benutzer suchen..."
			class="w-full bg-white border border-slate-200 rounded-xl py-2 px-3 pl-9 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all font-medium text-slate-800"
		/>
		<span class="absolute left-3 top-2.5 text-slate-400">🔍</span>
	</div>
	<button
		onclick={openNewUserModal}
		class="w-full sm:w-auto px-4 py-2 text-xs font-bold text-white bg-blue-600 hover:bg-blue-700 rounded-xl transition-all shadow-xs flex items-center justify-center gap-1.5 cursor-pointer"
	>
		➕ Benutzer anlegen
	</button>
</div>

<!-- User Table -->
<UserManagementTable {loadingUsers} {filteredUsers} {openEditUserModal} {openDeleteConfirm} />

<!-- Modal: Create / Edit User -->
<UserManagementEditModal
	open={showUserModal}
	onclose={() => (showUserModal = false)}
	{isEditingUser}
	bind:userForm
	{submittingUser}
	{error}
	{handleSaveUser}
/>

<!-- Modal: Delete Confirmation -->
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
