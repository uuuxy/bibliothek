// stores/mahnwesen.svelte.js
// Status- und Logikverwaltung für das Mahnwesen (Svelte 5 Runes)

import { apiFetch } from "../apiFetch.js";
import { useMahnwesenPdf } from "./mahnwesenPdf.svelte.js";
import { useMahnwesenMail } from "./mahnwesenMail.svelte.js";

/**
 * Creates the central Mahnwesen store.
 */
export function createMahnwesenStore() {
    let data = $state(/** @type {{ klassen: any[] } | null} */ (null));
    let loading = $state(true);
    let error = $state(/** @type {string|null} */ (null));

    // Ferien-Logik
    let ferienAktiv = $state(false);
    let ferienBezeichnung = $state("");
    let heuteRetourniert = $state(0);

    // Filter und Auswahl
    let expandedKlassen = $state(/** @type {Set<string>} */ (new Set()));
    let mahnMode = $state("datum"); // "datum" oder "jahrgang"
    let selectedKlasse = $state("");

    // MD3 Filter & Bulk Actions
    let activeFilter = $state("Alle"); // "Alle", "1. Erinnerung", "Mahnung", "Lehrerkollegium"
    let selectedIds = $state(/** @type {Set<string>} */ (new Set()));

    const pdfStore = useMahnwesenPdf();
    const mailStore = useMahnwesenMail();

    // Abgeleitete Werte
    let klassen = $derived(data?.klassen ?? []);
    let totalOverdue = $derived(
        klassen.reduce((/** @type {number} */ sum, /** @type {any} */ k) =>
            sum + k.schueler.reduce((/** @type {number} */ s2, /** @type {any} */ sch) => s2 + sch.medien.length, 0), 0)
    );

    // Flache Liste für MD3 Table
    let flatSchueler = $derived(() => {
        let list = [];
        for (const k of klassen) {
            for (const s of k.schueler) {
                let maxTage = 0;
                for (const m of s.medien) {
                    if (m.tage_ueberfaellig > maxTage) maxTage = m.tage_ueberfaellig;
                }
                
                let mahnstufe = "Erledigt";
                if (s.klasse && s.klasse.toLowerCase() === "lehrer") {
                    mahnstufe = "Lehrerkollegium";
                } else if (maxTage > 14) {
                    mahnstufe = "Mahnung";
                } else if (maxTage > 0) {
                    mahnstufe = "1. Erinnerung";
                }
                
                list.push({ ...s, maxTage, mahnstufe, lehrer_email: k.lehrer_email });
            }
        }
        return list;
    });

    let filteredSchueler = $derived(() => {
        const list = flatSchueler();
        if (activeFilter === "Alle") return list;
        return list.filter((/** @type {any} */ s) => s.mahnstufe === activeFilter);
    });

    /**
     * Fetches Mahnwesen data from API.
     */
    async function fetchData() {
        loading = true;
        error = null;
        try {
            const endpoint = mahnMode === "datum" ? "/api/mahnwesen" : "/api/mahnwesen/ueberfaellig_jahrgang";
            const res = await apiFetch(endpoint);
            if (!res.ok) throw new Error(await res.text() || "Fehler beim Laden");
            const json = await res.json();
            data = json;
            ferienAktiv = json.ferien_aktiv || false;
            ferienBezeichnung = json.ferien_bezeichnung || "";
            heuteRetourniert = json.heute_retourniert || 0;
            selectedIds.clear();
        } catch (e) {
            error = String(e);
        } finally {
            loading = false;
        }
    }

    /** @param {string} klasse */
    function toggleKlasse(klasse) {
        const s = new Set(expandedKlassen);
        if (s.has(klasse)) s.delete(klasse);
        else s.add(klasse);
        expandedKlassen = s;
    }

    /** @param {string} id */
    function toggleSelect(id) {
        const s = new Set(selectedIds);
        if (s.has(id)) s.delete(id);
        else s.add(id);
        selectedIds = s;
    }

    /** @param {boolean} selectAll */
    function toggleSelectAll(selectAll) {
        if (!selectAll) {
            selectedIds = new Set();
        } else {
            const currentList = filteredSchueler();
            selectedIds = new Set(currentList.map((/** @type {any} */ s) => s.schueler_id));
        }
    }

    /** Wrapper for bulk printing that supplies needed state context. */
    async function printSelectedMahnungenWrapper() {
        await pdfStore.printSelectedMahnungen(selectedIds, filteredSchueler, fetchData);
    }

    return {
        get data() { return data; },
        get loading() { return loading; },
        get error() { return error; },
        get expandedKlassen() { return expandedKlassen; },
        get mahnMode() { return mahnMode; },
        set mahnMode(v) { mahnMode = v; },
        get selectedKlasse() { return selectedKlasse; },
        set selectedKlasse(v) { selectedKlasse = v; },
        get klassen() { return klassen; },
        get totalOverdue() { return totalOverdue; },
        get ferienAktiv() { return ferienAktiv; },
        get ferienBezeichnung() { return ferienBezeichnung; },
        get heuteRetourniert() { return heuteRetourniert; },
        get activeFilter() { return activeFilter; },
        set activeFilter(v) { activeFilter = v; selectedIds.clear(); },
        get selectedIds() { return selectedIds; },
        get filteredSchueler() { return filteredSchueler(); },

        fetchData,
        toggleKlasse,
        toggleSelect,
        toggleSelectAll,
        
        // Expose mail store methods and state directly
        get modalOpen() { return mailStore.modalOpen; },
        get modalKlasse() { return mailStore.modalKlasse; },
        get modalEmail() { return mailStore.modalEmail; },
        set modalEmail(v) { mailStore.modalEmail = v; },
        get modalSending() { return mailStore.modalSending; },
        get modalMsg() { return mailStore.modalMsg; },
        openModal: mailStore.openModal,
        closeModal: mailStore.closeModal,
        sendMahnliste: mailStore.sendMahnliste,

        // Expose pdf store methods and state directly
        get pdfLoading() { return pdfStore.pdfLoading; },
        get elternPdfLoading() { return pdfStore.elternPdfLoading; },
        get klassePdfLoading() { return pdfStore.klassePdfLoading; },
        get globalErrorToast() { return pdfStore.globalErrorToast; },
        printSelectedMahnungen: printSelectedMahnungenWrapper,
        downloadPDF: pdfStore.downloadPDF,
        downloadElternPDF: pdfStore.downloadElternPDF,
        downloadKlassePDF: pdfStore.downloadKlassePDF
    };
}

export const mahnwesenStore = createMahnwesenStore();
