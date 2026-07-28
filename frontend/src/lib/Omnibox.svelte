<script>
	import { onMount } from 'svelte';
	import StudentProfile from './StudentProfile.svelte';
	import CameraScanner from './CameraScanner.svelte';
	import OmniboxInput from './components/OmniboxInput.svelte';
	import OmniboxResults from './components/OmniboxResults.svelte';
	import OmniboxTeacherCard from './OmniboxTeacherCard.svelte';
	import OmniboxVormerkungAlert from './components/OmniboxVormerkungAlert.svelte';
	import OmniboxBlockAlert from './components/OmniboxBlockAlert.svelte';
	import OmniboxScreenFlash from './components/OmniboxScreenFlash.svelte';
	import { omniboxStore } from './stores/omnibox.svelte.js';
	import { appState } from '../inventur/lib/store.svelte.js';

	let { onSelectBook, role = '' } = $props();

	let studentProfileComponent = $state(/** @type {any} */ (null));

	// Rückmeldung des Scanners (rot = blockiert/fehlgeschlagen, grün = gebucht, orange = Hinweis).
	//
	// Entscheidend ist, dass Grundzustand und Rückmeldung sich AUSSCHLIESSEN, statt
	// nebeneinander im class-Attribut zu stehen: Tailwind-Utilities haben alle dieselbe
	// Spezifität, es gewinnt die Regel, die im Stylesheet WEITER HINTEN steht — nicht die,
	// die im Attribut später kommt. `border-transparent` steht dort hinter `border-red-500`,
	// also hätte der Grundzustand die Fehlerfarbe geschluckt. (Genau so gemessen: Klassen
	// gesetzt, berechnete Rahmenfarbe trotzdem transparent.)
	//
	// Deshalb liefert dieser Ausdruck den KOMPLETTEN Farbsatz — entweder Ruhe oder Rückmeldung.
	// Ein Fokus-Blau während einer roten Rückmeldung gibt es damit gar nicht erst.
	const RUHE =
		'bg-slate-100 border-transparent focus-within:bg-white focus-within:border-blue-600 focus-within:ring-1 focus-within:ring-blue-600';
	const farbZustand = $derived.by(() => {
		if (omniboxStore.flashBorder === 'green') return 'bg-emerald-50 border-emerald-400 ring-1 ring-emerald-400';
		if (omniboxStore.flashBorder === 'orange') return 'bg-amber-50 border-amber-400 ring-1 ring-amber-400';
		if (omniboxStore.flashBorder === 'red') return 'bg-red-50 border-red-500 ring-1 ring-red-500';
		return RUHE;
	});

	$effect(() => {
		if (appState.triggerStudentScan) {
			omniboxStore.queryVal = appState.triggerStudentScan;
			appState.triggerStudentScan = '';
			omniboxStore.submitAction(null, () => studentProfileComponent?.reloadProfile());
		}
	});

	onMount(() => {
		// SSE für Live-Aktualisierung des Schülerprofils
		const source = new EventSource('/events');
		source.addEventListener('action', (e) => {
			try {
				const actionData = JSON.parse(/** @type {any} */ (e).data);
				if (omniboxStore.activeStudent && actionData.student_id === omniboxStore.activeStudent.id) {
					studentProfileComponent?.reloadProfile();
				}
			} catch (err) {
				console.error('SSE Parsing-Fehler in der Omnibox:', err);
			}
		});

		// Offline / Online Erkennung is now handled globally in offlineSync.svelte.js

		return () => {
			source.close();
		};
	});

	$effect(() => {
		if (!omniboxStore.isActive && !omniboxStore.isDropdownOpen && !omniboxStore.showCamera) {
			setTimeout(() => document.getElementById('omnibox-input')?.focus(), 50);
		}
	});

	$effect(() => {
		/** @param {KeyboardEvent} e */
		function handleKeyDown(e) {
			if (e.key === 'Escape') {
				omniboxStore.queryVal = '';
				omniboxStore.activeStudent = null;
				omniboxStore.activeTeacher = null;
				omniboxStore.lastFremdrueckgabe = null;
				omniboxStore.isDropdownOpen = false;
				if (omniboxStore.showCamera) {
					omniboxStore.showCamera = false;
					if (omniboxStore.cameraScanner) {
						try {
							omniboxStore.cameraScanner.stop();
						} catch {}
						try {
							omniboxStore.cameraScanner.clear();
						} catch {}
						omniboxStore.cameraScanner = null;
					}
					setTimeout(() => document.getElementById('omnibox-input')?.focus(), 50);
				}
			}
		}
		window.addEventListener('keydown', handleKeyDown);
		return () => window.removeEventListener('keydown', handleKeyDown);
	});

	// HTML5 Kamera-Scanner (Mobile)
	async function startCamera() {
		omniboxStore.showCamera = true;
		await new Promise((r) => setTimeout(r, 80));
		try {
			const { Html5Qrcode } = await import('html5-qrcode');
			omniboxStore.cameraScanner = new Html5Qrcode('camera-scan-region');
			await omniboxStore.cameraScanner.start(
				{ facingMode: 'environment' },
				{ fps: 10, qrbox: { width: 260, height: 120 } },
				(/** @type {string} */ decodedText) => {
					omniboxStore.queryVal = decodedText.trim();
					stopCamera();
					omniboxStore.submitAction(null, () => studentProfileComponent?.reloadProfile());
				},
				() => {}
			);
		} catch {
			omniboxStore.showCamera = false;
			omniboxStore.showToast('Kamera konnte nicht gestartet werden', 'error');
		}
	}

	async function stopCamera() {
		omniboxStore.showCamera = false;
		if (omniboxStore.cameraScanner) {
			try {
				await omniboxStore.cameraScanner.stop();
			} catch {}
			try {
				omniboxStore.cameraScanner.clear();
			} catch {}
			omniboxStore.cameraScanner = null;
		}
		setTimeout(() => document.getElementById('omnibox-input')?.focus(), 50);
	}
</script>

<OmniboxScreenFlash />

<!-- ── Offline / Queue banner was replaced by global OfflineIndicator ── -->

<div
	class="w-full mx-auto transition-all duration-500 ease-in-out {omniboxStore.isActive
		? 'w-full pt-4 justify-start'
		: 'max-w-2xl min-h-[60vh] justify-center'} flex flex-col items-center space-y-6"
>
	<div
		class="w-full transition-all duration-500 {omniboxStore.isActive
			? 'sticky -top-4 z-30 bg-slate-50/95 backdrop-blur-md py-4'
			: ''}"
	>
		<!-- Material-3-Suchleiste: weiche Pille mit Flächen-Fokus. Bewusst rounded-full und
		     bewusst 48 px statt der 36-px-Control-Höhe — das Scanfeld ist das globale Werkzeug
		     des Kiosks und soll sich von den eckigen Datenfeldern abheben. Der Container trägt
		     Fläche, Rahmen und Fokus; Lupe, Feld und Kamera-Knopf sind seine Flex-Kinder.
		     `relative` bleibt: die Ergebnisliste hängt sich mit top-full daran. -->
		<form
			onsubmit={(e) => omniboxStore.submitAction(e, () => studentProfileComponent?.reloadProfile())}
			class="group relative flex items-center w-full h-12 px-5 rounded-full border transition-all duration-200 focus-within:shadow-md no-print {omniboxStore.isShaking
				? 'animate-shake'
				: ''} {farbZustand}"
		>
			<OmniboxInput
				bind:queryVal={omniboxStore.queryVal}
				isDropdownOpen={omniboxStore.isDropdownOpen}
				totalDropdownItems={omniboxStore.totalDropdownItems}
				isActive={omniboxStore.isActive}
				showCamera={omniboxStore.showCamera}
				onInput={omniboxStore.handleInput}
				onSelect={(idx) => omniboxStore.selectDropdownItem(idx, onSelectBook)}
				onIndexChange={(idx) => (omniboxStore.selectedDropdownIndex = idx)}
				onEscape={() => (omniboxStore.isDropdownOpen = false)}
				onToggleCamera={omniboxStore.showCamera ? stopCamera : startCamera}
			/>

			{#if omniboxStore.isDropdownOpen && omniboxStore.totalDropdownItems > 0}
				<OmniboxResults
					unifiedSearchResults={omniboxStore.unifiedSearchResults}
					selectedDropdownIndex={omniboxStore.selectedDropdownIndex}
					onSelect={(idx) => omniboxStore.selectDropdownItem(idx, onSelectBook)}
				/>
			{/if}
		</form>

		{#if omniboxStore.errorMessage}
			<div
				class="mt-3 p-3 bg-red-600 text-white font-bold rounded-xl shadow-lg text-center animate-slide-down"
			>
				{omniboxStore.errorMessage}
			</div>
		{/if}
	</div>

	<!-- HTML5 Kamera-Scanner (Mobile) -->
	{#if omniboxStore.showCamera}
		<CameraScanner
			{stopCamera}
			bind:queryVal={omniboxStore.queryVal}
			submitAction={(e) =>
				omniboxStore.submitAction(e, () => studentProfileComponent?.reloadProfile())}
		/>
	{/if}

	{#if omniboxStore.activeStudent}
		{#if omniboxStore.lastFremdrueckgabe}
			<div
				class="w-full max-w-xl p-3 rounded-xl bg-amber-50 border border-amber-100 text-amber-800 text-xs font-medium flex items-center space-x-2 animate-slide-up no-print mb-2"
			>
				<span>⚠️</span>
				<span
					><strong>Fremdrückgabe:</strong> Buch war auf
					<strong>{omniboxStore.lastFremdrueckgabe.vorbesitzerName}</strong>
					verbucht und wurde dort zurückgegeben — <strong>nicht</strong> auf {omniboxStore
						.activeStudent.vorname} gebucht. Erneut scannen, um es auszuleihen.</span
				>
			</div>
		{/if}
		<StudentProfile
			bind:this={studentProfileComponent}
			student={omniboxStore.activeStudent}
			{role}
			onDeselect={() => {
				omniboxStore.activeStudent = null;
				omniboxStore.lastFremdrueckgabe = null;
			}}
			onReturnClick={(barcode) => {
				omniboxStore.queryVal = barcode;
				omniboxStore.submitAction(null, () => studentProfileComponent?.reloadProfile());
			}}
		/>
	{:else if omniboxStore.activeTeacher}
		<OmniboxTeacherCard
			teacher={omniboxStore.activeTeacher}
			onDeselect={() => {
				omniboxStore.activeTeacher = null;
				omniboxStore.lastFremdrueckgabe = null;
			}}
		/>
	{/if}
</div>

<!-- Toast notifications -->
<div
	class="fixed top-24 left-1/2 -translate-x-1/2 w-full max-w-lg z-50 space-y-3 px-4 pointer-events-none"
>
	{#if omniboxStore.toast}
		<div
			class="p-4 rounded-xl shadow-xl flex items-center space-x-3 backdrop-blur-md animate-slide-down pointer-events-auto border
      {omniboxStore.toast.type === 'success'
				? 'bg-emerald-50 border-emerald-100/50 text-emerald-700'
				: omniboxStore.toast.type === 'warning'
					? 'bg-amber-50 border-amber-100/50 text-amber-700'
					: 'bg-red-600 border-red-700 text-white shadow-red-500/30'}"
		>
			<span class="text-sm font-semibold">{omniboxStore.toast.message}</span>
		</div>
	{/if}
</div>

<OmniboxVormerkungAlert />

<OmniboxBlockAlert onReload={() => studentProfileComponent?.reloadProfile()} />

<style>
	/* ── Shake animation ─────────────────────────────────────── */
	@keyframes shake {
		0%,
		100% {
			transform: translate(0, 0) scale(1.05);
		}
		15%,
		45%,
		75% {
			transform: translate(-8px, 0) scale(1.05);
		}
		30%,
		60% {
			transform: translate(8px, 0) scale(1.05);
		}
	}
	@keyframes activeShake {
		0%,
		100% {
			transform: translate(0, 0) scale(1);
		}
		15%,
		45%,
		75% {
			transform: translate(-8px, 0) scale(1);
		}
		30%,
		60% {
			transform: translate(8px, 0) scale(1);
		}
	}
	.animate-shake {
		animation: shake 0.4s cubic-bezier(0.36, 0.07, 0.19, 0.97) both;
	}
	:global(.pt-4) .animate-shake {
		animation: activeShake 0.4s cubic-bezier(0.36, 0.07, 0.19, 0.97) both;
	}
</style>
