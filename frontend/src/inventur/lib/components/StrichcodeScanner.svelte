<script>
	import { apiFetch } from '../../../lib/apiFetch.js';
	import IsbnLookupDialog from "$lib/components/IsbnLookupDialog.svelte";
	import KameraScanner from "$lib/components/scanner/KameraScanner.svelte";
	import ManualInput from "$lib/components/scanner/ManualInput.svelte";
	import FileUploader from "$lib/components/scanner/FileUploader.svelte";

	let {
		subject = "Mathe",
		gradeLevel = 5,
		onClose = () => {},
		onCreated = () => {},
	} = $props();
	let status = $state("Bereit zum Scannen.");
	let scanning = $state(false);
	let busy = $state(false);
	let lastCode = $state("");
	/** @type {any} */
	let lookupData = $state(null);
	/** @type {any} */
	let cameraCmp = $state(null);

	/**
	 * @param {string} value
	 */
	function normalizeISBN(value) {
		return value.replace(/[^0-9Xx]/g, "").toUpperCase();
	}

	/**
	 * @param {string} msg
	 * @param {boolean} [isBusy]
	 */
	function handleStatusChange(msg, isBusy = false) {
		status = msg;
		if (isBusy !== undefined) busy = isBusy;
	}

	/**
	 * @param {string} raw
	 */
	async function submitISBN(raw) {
		const isbn = normalizeISBN(raw);
		if (isbn.length < 10 || isbn.length > 13 || isbn === lastCode || busy)
			return;
		lastCode = isbn;
		busy = true;
		status = `ISBN erkannt: ${isbn}`;

		if (cameraCmp) await cameraCmp.stopScanner();

		try {
			const res = await apiFetch(`/api/lookup/${isbn}`);
			if (!res.ok) throw new Error();
			const payload = await res.json();
			lookupData = payload.data;
			status = "Metadaten geladen. Bitte prüfen und speichern.";
		} catch {
			lookupData = {
				isbn,
				title: "",
				author: "",
				coverUrl: "",
				subject,
				grade: String(gradeLevel),
			};
			status = "Keine Metadaten gefunden. Bitte manuell ergänzen.";
		} finally {
			busy = false;
		}
	}

	/**
	 * @param {any} payload
	 */
	async function saveBook(payload) {
		busy = true;
		try {
			const res = await apiFetch("/api/books", {
				method: "POST",
				credentials: "include",
				headers: /** @type {HeadersInit} */ ({
					"Content-Type": "application/json",
				}),
				body: JSON.stringify(payload),
			});
			let errorMessage = "";
			if (!res.ok) {
				const body = await res.json().catch(() => null);
				errorMessage = body?.error ?? "";
			}
			if (res.status === 401) throw new Error("unauthorized");
			if (!res.ok) throw new Error(errorMessage || "request_failed");
			status = `Buch ${payload.isbn} wurde gespeichert.`;
			lookupData = null;
			lastCode = "";
			onCreated(payload.isbn);
			if (cameraCmp) await cameraCmp.startScanner();
		} catch (error) {
			const err = /** @type {any} */ (error);
			if (err?.message === "unauthorized") {
				status =
					"401 Unauthorized: Admin-Token ist falsch oder abgelaufen.";
			} else if (err?.message && err?.message !== "request_failed") {
				status = `Speichern fehlgeschlagen: ${err.message}`;
			} else {
				status =
					"Speichern fehlgeschlagen. Bitte Daten und API prüfen.";
			}
		} finally {
			busy = false;
		}
	}

	async function cancelLookup() {
		lookupData = null;
		status = "Erfassung abgebrochen.";
		lastCode = "";
		if (cameraCmp) await cameraCmp.startScanner();
	}

	function manualScannerClose() {
		if (cameraCmp) cameraCmp.stopScanner();
		onClose();
	}
</script>

<div
	class="relative w-full max-w-xl rounded-3xl bg-white border border-slate-200 p-6 shadow-2xl text-slate-800"
>
	<div class="mb-4 flex items-start justify-between gap-3">
		<div>
			<h3 class="text-lg font-bold text-slate-900">
				ISBN-Scanner
			</h3>
			<p class="mt-1.5 text-sm text-slate-500 font-medium">
				{status}
			</p>
		</div>
		<button
			onclick={manualScannerClose}
			class="rounded-xl bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-755 hover:bg-slate-200 hover:text-slate-900 transition-colors cursor-pointer"
			>Schließen</button
		>
	</div>

	<KameraScanner
		bind:this={cameraCmp}
		bind:scanning
		onDecode={submitISBN}
		onStatusChange={handleStatusChange}
		showControls={false}
	/>

	<ManualInput onSubmit={submitISBN} disabled={busy} />

	<div class="mt-4 flex gap-3">
		<button
			onclick={() => cameraCmp?.startScanner()}
			disabled={scanning || busy || !!lookupData}
			class="rounded-xl bg-blue-600 px-5 py-2.5 text-sm font-bold text-white hover:bg-blue-700 disabled:opacity-60 transition-colors cursor-pointer shadow-sm"
			>Starten</button
		>
		<button
			onclick={() => cameraCmp?.stopScanner()}
			disabled={!scanning}
			class="rounded-xl bg-slate-100 border border-slate-250 px-5 py-2.5 text-sm font-semibold text-slate-700 hover:bg-slate-200 disabled:opacity-60 transition-colors cursor-pointer"
			>Stoppen</button
		>
	</div>

	<FileUploader
		onDecode={submitISBN}
		onStatusChange={(/** @type {string} */ msg, /** @type {boolean} */ isBusy) => handleStatusChange(msg, isBusy)}
		disabled={busy}
	/>
</div>

{#if lookupData}
	<IsbnLookupDialog
		data={lookupData}
		{busy}
		onCancel={cancelLookup}
		onSave={saveBook}
	/>
{/if}
