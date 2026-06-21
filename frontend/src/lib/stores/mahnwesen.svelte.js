// stores/mahnwesen.svelte.js
// Status- und Logikverwaltung für das Mahnwesen (Svelte 5 Runes)

import { apiFetch, apiClient } from "../apiFetch.js";

export function createMahnwesenStore() {
    let data = $state(/** @type {{ klassen: any[] } | null} */ (null));
    let loading = $state(true);
    let error = $state(/** @type {string|null} */ (null));
    let globalErrorToast = $state(/** @type {string|null} */ (null));

    // Ferien-Logik
    let ferienAktiv = $state(false);
    let ferienBezeichnung = $state("");

    // Modal-Status
    let modalOpen = $state(false);
    let modalKlasse = $state("");
    let modalEmail = $state("");
    let modalSending = $state(false);
    let modalMsg = $state(/** @type {{ type: 'success'|'error', text: string }|null} */ (null));

    // Ladezustände für PDFs
    let pdfLoading = $state(false);
    let elternPdfLoading = $state(false);
    let klassePdfLoading = $state(false);

    // Filter und Auswahl
    let expandedKlassen = $state(/** @type {Set<string>} */ (new Set()));
    let mahnMode = $state("datum"); // "datum" oder "jahrgang"
    let selectedKlasse = $state("");

    // Individuelle Schüler-Mails
    let sendingStudentId = $state(/** @type {string|null} */ (null));
    let studentMessages = $state(/** @type {Record<string, {type: 'success'|'error', text: string} | null>} */ ({}));

    // MD3 Filter & Bulk Actions
    let activeFilter = $state("Alle"); // "Alle", "1. Erinnerung", "Mahnung", "Lehrerkollegium"
    let selectedIds = $state(/** @type {Set<string>} */ (new Set()));

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
        return list.filter(s => s.mahnstufe === activeFilter);
    });

    // Methoden
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
            selectedIds.clear(); // Reset selection on new data
        } catch (e) {
            error = String(e);
        } finally {
            loading = false;
        }
    }

    /** @param {string} schuelerId */
    async function sendStudentMahnung(schuelerId) {
        sendingStudentId = schuelerId;
        studentMessages[schuelerId] = null;
        try {
            const res = await apiFetch(`/api/mail/send-overdue-notification/${schuelerId}`, { method: "POST" });
            const json = await res.json();
            if (res.ok) {
                studentMessages[schuelerId] = { type: 'success', text: json.message || "Gesendet" };
            } else {
                studentMessages[schuelerId] = { type: 'error', text: json.error || json.message || "Fehler" };
            }
        } catch (e) {
            studentMessages[schuelerId] = { type: 'error', text: String(e) };
        } finally {
            sendingStudentId = null;
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

    async function printSelectedMahnungen() {
        if (selectedIds.size === 0) return;
        
        pdfLoading = true;
        try {
            // Sammle alle Ausleih-IDs der selektierten Schüler
            const currentList = filteredSchueler();
            const ausleihIds = [];
            for (const schuelerId of selectedIds) {
                const s = currentList.find((/** @type {any} */ x) => x.schueler_id === schuelerId);
                if (s && s.medien) {
                    for (const m of s.medien) {
                        if (m.ausleihe_id) {
                            ausleihIds.push(m.ausleihe_id);
                        }
                    }
                }
            }

            if (ausleihIds.length === 0) {
                alert("Keine überfälligen Medien für die ausgewählten Schüler gefunden.");
                pdfLoading = false;
                return;
            }

            const res = await apiFetch("/api/admin/mahnungen/bulk-print", {
                method: "POST",
                body: JSON.stringify({ ausleih_ids: ausleihIds })
            });

            if (!res.ok) throw new Error("Bulk-PDF-Erzeugung fehlgeschlagen: " + (await res.text() || res.statusText));
            
            const blob = await res.blob();
            const url = URL.createObjectURL(blob);
            const a = document.createElement("a");
            a.href = url;
            a.download = `Mahnliste_Bulk_${new Date().toISOString().slice(0,10)}.pdf`;
            a.click();
            URL.revokeObjectURL(url);

            selectedIds.clear();
            await fetchData(); // Liste neu laden, um die aktualisierten Mahnstufen zu sehen
        } catch (e) {
            alert("Fehler: " + String(e));
        } finally {
            pdfLoading = false;
        }
    }

    async function downloadPDF() {
        pdfLoading = true;
        try {
            const res = await apiFetch("/api/mahnwesen/pdf");
            if (!res.ok) throw new Error("PDF-Erzeugung fehlgeschlagen");
            const blob = await res.blob();
            const url = URL.createObjectURL(blob);
            const a = document.createElement("a");
            a.href = url;
            a.download = `mahnliste_${new Date().toISOString().slice(0,10)}.pdf`;
            a.click();
            URL.revokeObjectURL(url);
        } catch (e) {
            alert("Fehler: " + String(e));
        } finally {
            pdfLoading = false;
        }
    }

    async function downloadElternPDF() {
        elternPdfLoading = true;
        globalErrorToast = null;
        try {
            const res = await apiFetch("/api/reports/overdue-pdf");
            if (!res.ok) throw new Error(await res.text() || "PDF-Erzeugung fehlgeschlagen");
            const blob = await res.blob();
            const url = URL.createObjectURL(blob);
            const a = document.createElement("a");
            a.href = url;
            a.download = `mahnbriefe_${new Date().toISOString().slice(0,10)}.pdf`;
            a.click();
            URL.revokeObjectURL(url);
        } catch (e) {
            globalErrorToast = "Fehler: " + String(e);
            setTimeout(() => globalErrorToast = null, 4000);
        } finally {
            elternPdfLoading = false;
        }
    }

    async function downloadKlassePDF() {
        if (!selectedKlasse) return;
        klassePdfLoading = true;
        globalErrorToast = null;
        try {
            const res = await apiFetch(`/api/print/mahnung/klasse/${selectedKlasse}`);
            if (!res.ok) {
                const errText = await res.text();
                throw new Error(errText || "Keine überfälligen Ausleihen gefunden");
            }
            const blob = await res.blob();
            const url = URL.createObjectURL(blob);
            const a = document.createElement("a");
            a.href = url;
            a.download = `Mahnliste_Klasse_${selectedKlasse}.pdf`;
            a.click();
            URL.revokeObjectURL(url);
        } catch (e) {
            globalErrorToast = String(e);
            setTimeout(() => globalErrorToast = null, 4000);
        } finally {
            klassePdfLoading = false;
        }
    }

    /**
     * @param {string} klasse
     * @param {string|null} [email]
     */
    function openModal(klasse, email) {
        modalKlasse = klasse;
        modalEmail = email ?? "";
        modalMsg = null;
        modalOpen = true;
    }

    function closeModal() {
        modalOpen = false;
        modalKlasse = "";
        modalEmail = "";
        modalMsg = null;
    }

    async function sendMahnliste() {
        if (!modalEmail.trim()) { modalMsg = { type: 'error', text: 'E-Mail-Adresse angeben.' }; return; }
        modalSending = true;
        modalMsg = null;
        try {
            const res = await apiFetch("/api/mahnwesen/senden", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ klasse: modalKlasse, email: modalEmail }),
            });
            const json = await res.json();
            if (res.ok) {
                modalMsg = { type: 'success', text: json.message ?? "Gesendet." };
            } else {
                modalMsg = { type: 'error', text: json.error ?? json.message ?? "Fehler." };
            }
        } catch (e) {
            modalMsg = { type: 'error', text: String(e) };
        } finally {
            modalSending = false;
        }
    }

    return {
        get data() { return data; },
        get loading() { return loading; },
        get error() { return error; },
        get globalErrorToast() { return globalErrorToast; },
        get modalOpen() { return modalOpen; },
        get modalKlasse() { return modalKlasse; },
        get modalEmail() { return modalEmail; },
        set modalEmail(v) { modalEmail = v; },
        get modalSending() { return modalSending; },
        get modalMsg() { return modalMsg; },
        get pdfLoading() { return pdfLoading; },
        get elternPdfLoading() { return elternPdfLoading; },
        get klassePdfLoading() { return klassePdfLoading; },
        get expandedKlassen() { return expandedKlassen; },
        get mahnMode() { return mahnMode; },
        set mahnMode(v) { mahnMode = v; },
        get selectedKlasse() { return selectedKlasse; },
        set selectedKlasse(v) { selectedKlasse = v; },
        get sendingStudentId() { return sendingStudentId; },
        get studentMessages() { return studentMessages; },
        get klassen() { return klassen; },
        get totalOverdue() { return totalOverdue; },
        get ferienAktiv() { return ferienAktiv; },
        get ferienBezeichnung() { return ferienBezeichnung; },
        get activeFilter() { return activeFilter; },
        set activeFilter(v) { activeFilter = v; selectedIds.clear(); },
        get selectedIds() { return selectedIds; },
        get filteredSchueler() { return filteredSchueler(); },

        fetchData,
        sendStudentMahnung,
        toggleKlasse,
        toggleSelect,
        toggleSelectAll,
        printSelectedMahnungen,
        downloadPDF,
        downloadElternPDF,
        downloadKlassePDF,
        openModal,
        closeModal,
        sendMahnliste
    };
}

export const mahnwesenStore = createMahnwesenStore();
