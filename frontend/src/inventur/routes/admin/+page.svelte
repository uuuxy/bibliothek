<!--
  admin/+page.svelte
  Hauptseite für den Administratorenbereich der Inventur-App.
  Liest und schreibt Bücherdaten mithilfe der API und steuert Unterkomponenten.
-->
<script>
	import { onMount } from "svelte";
	import { appState } from "$lib/store.svelte.js";
	import BookTable from "$lib/components/admin/BookTable.svelte";
	import BuchFormular from "$lib/components/admin/BuchFormular.svelte";
	import StrichcodeScanner from "$lib/components/StrichcodeScanner.svelte";
	import KlassenUebersicht from "$lib/components/admin/KlassenUebersicht.svelte";
	import AdminBuchAktionen from "$lib/components/admin/AdminBuchAktionen.svelte";
	import AdminAnsichtsUmschalter from "$lib/components/admin/AdminAnsichtsUmschalter.svelte";
	import {
		holeBuecherListe,
		importiereExcel,
		loescheBuecher,
		holeExterneCover,
		retryExterneCover,
	} from "$lib/admin_api.js";

	/** @type {any[]} */
	let buecher = $state([]);
	let wirdGeladen = $state(false);
	let istBearbeitenModus = $state(false);
	let wirdGescannt = $state(false);
	let ansichtsModus = $state("list"); // 'list' | 'classes'
	let buchAktionen = $state();

	let formular = $state({
		id: null,
		isbn: "",
		title: "",
		author: "",
		subject: "Mathe",
		gradeLevel: 5,
		track: "Gymnasium",
		stock: 0,
		coverUrl: "",
		lastCounted: "",
		medientyp: "Buch",
	});

	/** @type {any} */
	let suchVerzoegerung = null;
	$effect(() => {
		const suchAnfrage = appState.searchQuery;
		if (suchVerzoegerung) clearTimeout(suchVerzoegerung);
		suchVerzoegerung = setTimeout(() => {
			if (appState.adminAuthenticated && typeof suchAnfrage === "string") {
				aktualisiereBuecher();
			}
		}, 300);
	});

	onMount(() => {
		aktualisiereBuecher();
	});

	async function aktualisiereBuecher() {
		wirdGeladen = true;
		try {
			const geladene = await holeBuecherListe();
			buecher = geladene;
			appState.adminAuthenticated = true;
		} catch {
			appState.adminAuthenticated = false;
		} finally {
			wirdGeladen = false;
		}
	}

	function neuesBuchErstellen() {
		formular = {
			id: null,
			isbn: "",
			title: "",
			author: "",
			subject: "Mathe",
			gradeLevel: 5,
			track: "Gymnasium",
			stock: 0,
			coverUrl: "",
			lastCounted: "",
			medientyp: "Buch",
		};
		istBearbeitenModus = true;
	}

	/** @param {any} buch */
	function oeffneDetails(buch) {
		formular = { ...buch };
		if (!formular.medientyp) {
			formular.medientyp = "Buch";
		}
		if (formular.lastCounted && formular.lastCounted.includes("T")) {
			formular.lastCounted = formular.lastCounted.split("T")[0];
		}
		istBearbeitenModus = true;
	}

	/** @param {any} ereignis */
	async function aktionExcelImport(ereignis) {
		const datei = ereignis.target.files[0];
		if (!datei) return;
		wirdGeladen = true;
		try {
			await importiereExcel(datei);
			await aktualisiereBuecher();
			alert("Import erfolgreich!");
		} catch (fehler) {
			alert(/** @type {any} */ (fehler).message);
		} finally {
			wirdGeladen = false;
		}
	}

	/** @param {any} ids */
	async function aktionBuecherLoeschen(ids) {
		if (!ids.length || !confirm(`${ids.length} Bücher wirklich löschen?`))
			return;
		try {
			await loescheBuecher(ids);
			buecher = buecher.filter((b) => !ids.includes(b.id));
		} catch (fehler) {
			alert(/** @type {any} */ (fehler).message);
		}
	}

	async function aktionExterneCoverRetry() {
		try {
			const externe = await holeExterneCover();
			if (!externe.length) {
				alert("Keine externen Cover mehr vorhanden.");
				return;
			}
			if (!confirm(`${externe.length} externe Cover jetzt erneut lokalisieren?`)) {
				return;
			}

			const ids = externe.map((/** @type {any} */ b) => b.id);
			const ergebnis = await retryExterneCover(ids);
			await aktualisiereBuecher();
			alert(`Cover-Retry fertig. Aktualisiert: ${ergebnis.updated}, Übersprungen: ${ergebnis.skipped}, Fehler: ${ergebnis.failed}`);
		} catch (fehler) {
			alert(/** @type {any} */ (fehler).message);
		}
	}

	function nachScanAktion() {
		aktualisiereBuecher();
	}
</script>

<div class="relative min-h-[calc(100vh-8rem)]">
	<AdminAnsichtsUmschalter
		{ansichtsModus}
		onWechsel={(modus) => (ansichtsModus = modus)}
	/>

	{#if wirdGescannt}
		<div
			class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4"
		>
			<StrichcodeScanner
				onClose={() => (wirdGescannt = false)}
				onCreated={nachScanAktion}
			/>
		</div>
	{:else if ansichtsModus === "classes"}
		<KlassenUebersicht />
	{:else}
		<BookTable
			books={buecher}
			loading={wirdGeladen}
			onOpenDetail={oeffneDetails}
			onImportExcel={aktionExcelImport}
			onCreateNew={neuesBuchErstellen}
			onScan={() => (wirdGescannt = true)}
			onDelete={aktionBuecherLoeschen}
			onRetryCovers={aktionExterneCoverRetry}
		/>
	{/if}

	<AdminBuchAktionen
		bind:this={buchAktionen}
		bind:books={buecher}
		bind:isEditMode={istBearbeitenModus}
		bind:formular
	/>

	{#if istBearbeitenModus}
		<BuchFormular
			bind:formular
			onClose={() => (istBearbeitenModus = false)}
			onSave={() => buchAktionen.saveChanges()}
			onCoverUpload={(/** @type {any} */ ereignis) =>
				buchAktionen.handleCoverUpload(ereignis)}
		/>
	{/if}
</div>
