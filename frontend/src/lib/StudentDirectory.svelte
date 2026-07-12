<script>
	import { apiFetch } from './apiFetch.js';
	import { onMount } from 'svelte';
	import StudentProfile from './StudentProfile.svelte';
	import StudentCreateModal from './StudentCreateModal.svelte';
	import Graduates from './Graduates.svelte';
	import ActiveStudentList from './components/students/ActiveStudentList.svelte';
	import DeletedStudentList from './components/students/DeletedStudentList.svelte';
	import StudentDirectoryToolbar from './components/students/StudentDirectoryToolbar.svelte';
	import PageContainer from './components/layout/PageContainer.svelte';

	// Props (Svelte 5)
	let { role = '' } = $props();

	// State Runes (Svelte 5)
	let activeTab = $state('active');

	/** @type {any[]} */
	let students = $state.raw([]);
	let loading = $state(false);
	let searchQuery = $state('');
	/** @type {any} */
	let activeStudent = $state(null);

	/** @type {any[]} */
	let readerGroups = $state.raw([]);
	let showCreateModal = $state(false);

	// Derived: client-seitig gefilterte Schülerliste
	let filteredStudents = $derived.by(() => {
		const q = searchQuery.toLowerCase().trim();
		if (!q) return students;
		return students.filter(
			(s) =>
				(s.vorname + ' ' + s.nachname).toLowerCase().includes(q) ||
				s.klasse.toLowerCase().includes(q) ||
				s.barcode_id.toLowerCase().includes(q)
		);
	});

	async function loadStudents() {
		loading = true;
		try {
			const res = await apiFetch('/api/schueler');
			if (res.ok) {
				students = await res.json();
			}
		} catch (err) {
			console.error('Fehler beim Laden des Schülerverzeichnisses:', err);
		} finally {
			loading = false;
		}
	}

	async function loadClasses() {
		try {
			const res = await apiFetch('/api/readergroups');
			if (res.ok) {
				readerGroups = (await res.json()) || [];
			}
		} catch (err) {
			console.error('Fehler beim Laden der Lesergruppen:', err);
		}
	}

	function handleStudentCreated() {
		showCreateModal = false;
		loadStudents();
		loadClasses(); // Klassenliste aktualisieren
	}

	onMount(() => {
		loadStudents();
		loadClasses();
	});
</script>

<div class="w-full h-full flex flex-col text-slate-800 bg-slate-50">
	{#if activeStudent}
		<div class="animate-fade-in flex-1 overflow-y-auto">
			<StudentProfile
				student={activeStudent}
				{role}
				onDeselect={() => {
					activeStudent = null;
					loadStudents();
				}}
			/>
		</div>
	{:else}
		<!-- Tab Navigation Header -->
		<div class="px-8 pt-6 pb-0 border-b border-slate-200 bg-white shrink-0 shadow-sm z-10">
			<div class="max-w-6xl mx-auto flex gap-6">
				{#snippet tabButton(id, label, activeColorClass)}
					<button
						onclick={() => (activeTab = id)}
						class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === id
							? activeColorClass
							: 'border-transparent text-slate-500 hover:text-slate-800'}"
					>
						{label}
					</button>
				{/snippet}

				{@render tabButton('active', 'Aktive Schüler', 'border-blue-600 text-blue-700')}
				{@render tabButton('graduates', 'Abgänger / Archiv', 'border-blue-600 text-blue-700')}
				{#if role === 'admin'}
					{@render tabButton('deleted', 'Papierkorb', 'border-rose-600 text-rose-700')}
				{/if}
			</div>
		</div>

		<!-- Tab Content -->
		<div class="flex-1 overflow-y-auto py-8 w-full">
			<PageContainer>
				{#if activeTab === 'active'}
					<div class="w-full no-print animate-fade-in">
						<StudentDirectoryToolbar
							bind:searchQuery
							{role}
							totalCount={students.length}
							filteredCount={filteredStudents.length}
							oncreate={() => (showCreateModal = true)}
						/>

						<div class="mt-6">
							<ActiveStudentList
								{filteredStudents}
								{students}
								{loading}
								onSelectStudent={(s) => (activeStudent = s)}
							/>
						</div>
					</div>
				{:else if activeTab === 'graduates'}
					<div class="w-full animate-fade-in">
						<Graduates />
					</div>
				{:else if activeTab === 'deleted'}
					<div class="w-full animate-fade-in space-y-6">
						<DeletedStudentList
							onRestoreSuccess={() => {
								loadStudents();
								loadClasses();
							}}
						/>
					</div>
				{/if}
			</PageContainer>
		</div>
	{/if}
</div>

<StudentCreateModal
	open={showCreateModal}
	{readerGroups}
	onclose={() => (showCreateModal = false)}
	onsuccess={handleStudentCreated}
/>
