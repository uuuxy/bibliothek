// stores/labels.svelte.js
// Status- und Logikverwaltung für den Etikettendruck (Svelte 5 Runes)

import { apiFetch } from "../apiFetch.js";
import { appState } from "../../inventur/lib/store.svelte.js";

export function createLabelStore() {
    let searchVal = $state("");
    let searchResults = $state(/** @type {any[]} */ ([]));
    let isSearching = $state(false);

    let classGroups = $state(/** @type {any[]} */ ([]));
    let selectedClass = $state("");
    let classBooks = $state(/** @type {any[]} */ ([]));

    let selectedTitle = $state(/** @type {any} */ (null));
    let barcodeType = $state("code39"); // "code39" | "qr"
    let labelBorder = $state(true);
    let startPosition = $state(1); // 1 to 21

    let generationMode = $state("existing");
    let existingCopies = $state(/** @type {any[]} */ ([]));
    let loadingCopies = $state(false);
    let newQuantity = $state(9);
    let newStartNum = $state(20060);

    let searchTimeout = /** @type {any} */ (null);

    /** @type {Array<{isBlank?: boolean, barcode_id?: string, titel?: string, autor?: string}>} */
    let finalLabels = $derived.by(() => {
        if ((appState.pendingPrintCopies?.length ?? 0) > 0) {
            const copies = /** @type {any[]} */ (appState.pendingPrintCopies);
            const rawList = copies.map(c => ({
                barcode_id: c.barcode_id,
                titel: c.titel,
                autor: c.autor || ""
            }));
            const offsetCount = Math.max(0, startPosition - 1);
            const offsetLabels = Array.from({ length: offsetCount }, () => ({ isBlank: true }));
            return [...offsetLabels, ...rawList];
        }

        if (!selectedTitle) return [];

        let rawList = [];
        if (generationMode === "existing") {
            rawList = existingCopies
                .filter(c => c.checked)
                .map(c => ({
                    barcode_id: c.barcode_id,
                    titel: selectedTitle.titel,
                    autor: selectedTitle.autor || ""
                }));
        } else {
            rawList = Array.from({ length: Math.max(1, newQuantity) }, (_, i) => ({
                barcode_id: `B-${newStartNum + i}`,
                titel: selectedTitle.titel,
                autor: selectedTitle.autor || ""
            }));
        }

        const offsetCount = Math.max(0, startPosition - 1);
        const offsetLabels = Array.from({ length: offsetCount }, () => ({ isBlank: true }));

        return [...offsetLabels, ...rawList];
    });

    async function loadClassGroups() {
        try {
            const res = await apiFetch("/api/class-books");
            if (res.ok) {
                const body = await res.json();
                if (body && body.data) {
                    classGroups = body.data;
                }
            }
        } catch (err) {
            console.error("Fehler beim Laden der Klassengruppen:", err);
        }
    }

    function handleClassChange() {
        const group = classGroups.find(g => g.className === selectedClass);
        if (group) {
            classBooks = group.books || [];
        } else {
            classBooks = [];
        }
        selectedTitle = null;
        existingCopies = [];
    }

    function handleSearchInput() {
        if (searchTimeout) clearTimeout(searchTimeout);
        if (!searchVal.trim()) {
            searchResults = [];
            return;
        }
        isSearching = true;
        searchTimeout = setTimeout(async () => {
            try {
                const res = await apiFetch("/api/action", {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ query: searchVal.trim() })
                });
                if (res.ok) {
                    const body = await res.json();
                    if (body.type === "search_results") {
                        searchResults = body.search_results || [];
                    }
                }
            } catch (err) {
                console.error("Fehler bei Buchtitelsuche:", err);
            } finally {
                isSearching = false;
            }
        }, 300);
    }

    /** @param {any} titleObj */
    async function selectBookTitle(titleObj) {
        selectedTitle = titleObj;
        searchResults = [];
        searchVal = titleObj.titel;
        selectedClass = "";
        classBooks = [];
        await loadExistingCopies();
    }

    async function loadExistingCopies() {
        if (!selectedTitle) return;
        loadingCopies = true;
        try {
            const res = await apiFetch(`/api/buecher/titel/${selectedTitle.id}/exemplare`);
            if (res.ok) {
                const data = await res.json();
                existingCopies = (data || []).map((/** @type {any} */ c) => ({
                    ...c,
                    checked: true
                }));
            } else {
                existingCopies = [];
            }
        } catch (err) {
            console.error("Fehler beim Laden der Exemplare:", err);
            existingCopies = [];
        } finally {
            loadingCopies = false;
        }
    }

    function triggerPrint() {
        window.print();
    }

    return {
        get searchVal() { return searchVal; },
        set searchVal(v) { searchVal = v; },
        get searchResults() { return searchResults; },
        get isSearching() { return isSearching; },
        
        get classGroups() { return classGroups; },
        get selectedClass() { return selectedClass; },
        set selectedClass(v) { selectedClass = v; },
        get classBooks() { return classBooks; },
        
        get selectedTitle() { return selectedTitle; },
        get barcodeType() { return barcodeType; },
        set barcodeType(v) { barcodeType = v; },
        get labelBorder() { return labelBorder; },
        set labelBorder(v) { labelBorder = v; },
        get startPosition() { return startPosition; },
        set startPosition(v) { startPosition = v; },
        
        get generationMode() { return generationMode; },
        set generationMode(v) { generationMode = v; },
        get existingCopies() { return existingCopies; },
        get loadingCopies() { return loadingCopies; },
        get newQuantity() { return newQuantity; },
        set newQuantity(v) { newQuantity = v; },
        get newStartNum() { return newStartNum; },
        set newStartNum(v) { newStartNum = v; },
        
        get finalLabels() { return finalLabels; },

        loadClassGroups,
        handleClassChange,
        handleSearchInput,
        selectBookTitle,
        triggerPrint,

        resetPendingCopies() {
            appState.pendingPrintCopies = null;
        }
    };
}

export const labelStore = createLabelStore();
