// stores/kiosk.svelte.js
// Status- und Logikverwaltung für den Kiosk-Modus (Svelte 5 Runes)

import { apiFetch } from "../apiFetch.js";
import { enqueueOfflineScan } from "../offlineQueue.js";
import { playSuccessBeep, playErrorBeep } from "../audio.js";
import { tick } from "svelte";

export function createKioskStore() {
    // ── Basis-Status ──────────────────────────────────────────────────
    let activeStudent = $state(/** @type {any} */ (null));
    let studentInputVal = $state("");
    let bookInputVal = $state("");
    let scannedBooks = $state(/** @type {any[]} */ ([]));

    // ── UI / Feedback ─────────────────────────────────────────────────
    let toast = $state(/** @type {any} */ (null));
    let screenFlash = $state(""); // "success" | "error" | "warning" | ""
    let isShaking = $state(false);
    let isScanningStudent = $state(false);
    let isScanningBook = $state(false);

    // ── Damage Modal State ────────────────────────────────────────────
    let returnedBook = $state(/** @type {any} */ (null));
    let returnedLoanId = $state("");
    let showDamageInput = $state(false);
    let damageDescription = $state("");
    let isSubmittingDamage = $state(false);

    // ── Vormerken Modal State ─────────────────────────────────────────
    let showVormerkenModal = $state(false);
    let vormerkenQuery = $state("");
    let vormerkenResults = $state(/** @type {any[]} */ ([]));
    let isSearchingVormerken = $state(false);
    let isSubmittingVormerken = $state(false);

    // ── Geraete Checklist Modal State ─────────────────────────────────
    let showChecklistModal = $state(false);
    let pendingGeraet = $state(/** @type {any} */ (null));
    let pendingGeraetQuery = $state("");
    let checklistItems = $derived.by(() => {
        if (!pendingGeraet || !pendingGeraet.zubehoer) return [];
        return pendingGeraet.zubehoer.split(",").map((/** @type {string} */ i) => i.trim()).filter(Boolean);
    });
    let checkedItems = $state(/** @type {Set<string>} */ (new Set()));
    let isSubmittingChecklist = $state(false);

    // ── Abgeleitete Werte & Settings ──────────────────────────────────
    let systemSettings = $state(/** @type {any} */ ({ max_ausleihen_schueler: 5 }));

    let activeLoansCount = $derived(activeStudent?.active_loans?.length || 0);
    let isLimitReached = $derived(activeLoansCount >= systemSettings.max_ausleihen_schueler);

    let isStudentBlocked = $derived.by(() => {
        if (!activeStudent) return false;
        const now = new Date().getTime();
        return activeStudent.active_loans?.some((/** @type {any} */ loan) => {
            if (!loan.rueckgabe_frist) return false;
            const frist = new Date(loan.rueckgabe_frist).getTime();
            return now > frist;
        }) ?? false;
    });

    // ── UI Helfer ─────────────────────────────────────────────────────
    /** @param {"success"|"error"|"warning"} type */
    function triggerFlash(type, msg = "") {
        screenFlash = type;
        if (type === "error") {
            isShaking = true;
            playErrorBeep();
            setTimeout(() => isShaking = false, 500);
        } else {
            playSuccessBeep();
        }
        if (msg) toast = { type, message: msg };
        setTimeout(() => { screenFlash = ""; }, 500);
        setTimeout(() => { toast = null; }, 4000);
    }

    function clearSession() {
        activeStudent = null;
        scannedBooks = [];
        studentInputVal = "";
        bookInputVal = "";
        focusStudentInput();
    }

    function focusStudentInput() {
        tick().then(() => document.getElementById("kiosk-student-input")?.focus());
    }

    function focusBookInput() {
        tick().then(() => document.getElementById("kiosk-book-input")?.focus());
    }

    // ── Daten laden ───────────────────────────────────────────────────
    async function fetchSettings() {
        try {
            const res = await apiFetch("/api/einstellungen");
            if (res.ok) {
                systemSettings = await res.json();
            }
        } catch(e) {}
    }

    // ── Aktionen ──────────────────────────────────────────────────────
    async function handleStudentSubmit() {
        const val = studentInputVal.trim();
        if (!val) return;
        isScanningStudent = true;
        try {
            const res = await apiFetch("/api/action", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ query: val })
            });
            if (!res.ok) throw new Error(await res.text());
            const data = await res.json();
            if (data.type === "student") {
                activeStudent = data.student;
                scannedBooks = [];
                triggerFlash("success");
                if (isStudentBlocked) {
                    triggerFlash("error", "Ausleihsperre! Überfällige Mahnungen vorhanden.");
                } else {
                    focusBookInput();
                }
            } else {
                throw new Error("Barcode ist kein Schülerausweis.");
            }
        } catch (e) {
            triggerFlash("error", e instanceof Error ? e.message : "Schüler nicht gefunden.");
            studentInputVal = "";
            focusStudentInput();
        } finally {
            isScanningStudent = false;
        }
    }

    async function handleBookSubmit() {
        const val = bookInputVal.trim();
        bookInputVal = "";
        if (!val || !activeStudent || isStudentBlocked) return;
        
        isScanningBook = true;
        try {
            const res = await apiFetch("/api/action", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ query: val, active_student_id: activeStudent.id })
            });
            if (!res.ok) throw new Error(await res.text());
            const data = await res.json();
            if (data.type === "ausleihe") {
                scannedBooks = [data.book, ...scannedBooks];
                triggerFlash("success");
                if (data.book.zustand_notiz) {
                    toast = { type: "error", message: `Achtung: Bekannter Mangel: ${data.book.zustand_notiz}` };
                }
            } else if (data.type === "rueckgabe") {
                if (data.has_vormerkung) {
                    triggerFlash("error");
                    toast = { type: "error", message: `ACHTUNG: Reserviert für ${data.vormerkung_user || 'eine/n Schüler/in'}! Bitte gesondert zurücklegen.` };
                    playErrorBeep();
                    setTimeout(playErrorBeep, 400);
                    returnedBook = null;
                } else {
                    returnedBook = data.book || data.geraet;
                    returnedLoanId = data.loan_id || (data.loanID ? data.loanID : "");
                    showDamageInput = false;
                    damageDescription = "";
                    playSuccessBeep();
                }
            } else if (data.type === "geraet_check") {
                pendingGeraet = data.geraet;
                pendingGeraetQuery = val;
                checkedItems = new Set();
                showChecklistModal = true;
            } else {
                throw new Error("Unerwartete Antwort vom Server.");
            }
        } catch (err) {
            const e = /** @type {any} */ (err);
            if (e instanceof TypeError && (e.message.includes("Failed to fetch") || e.message.includes("NetworkError"))) {
                await enqueueOfflineScan(val, activeStudent.id, null);
                triggerFlash("warning", "Offline: Scan gespeichert. Wird synchronisiert, sobald das Netzwerk wieder da ist.");
            } else {
                triggerFlash("error", e instanceof Error ? e.message : "Fehler beim Buchen.");
            }
            focusBookInput();
        } finally {
            isScanningBook = false;
        }
    }

    function handleDamageOk() {
        returnedBook = null;
        triggerFlash("success", "Buch zurückgegeben!");
        focusBookInput();
    }

    async function handleDamageSubmit() {
        if (!damageDescription.trim() || !returnedBook) return;
        isSubmittingDamage = true;
        try {
            const res = await apiFetch(`/api/buecher/exemplare/${returnedBook.id}/defekt`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ 
                    loan_id: returnedLoanId || undefined, 
                    schueler_id: activeStudent?.id || undefined,
                    betrag: 0,
                    beschreibung: damageDescription.trim()
                })
            });
            if (!res.ok) throw new Error(await res.text());
            triggerFlash("success", "Mangel gespeichert! Exemplar gesperrt.");
            returnedBook = null;
        } catch(e) {
            triggerFlash("error", e instanceof Error ? e.message : "Fehler beim Speichern des Mangels");
        } finally {
            isSubmittingDamage = false;
            focusBookInput();
        }
    }

    async function handleVormerkenSearch() {
        if (!vormerkenQuery.trim()) return;
        isSearchingVormerken = true;
        try {
            const res = await apiFetch("/api/action", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ query: vormerkenQuery })
            });
            if (!res.ok) throw new Error("Fehler bei der Suche");
            const data = await res.json();
            vormerkenResults = data.search_results || [];
        } catch(e) {
            triggerFlash("error", "Suche fehlgeschlagen");
        } finally {
            isSearchingVormerken = false;
        }
    }

    /** @param {string} titelId */
    async function handleVormerkenSubmit(titelId) {
        if (!activeStudent) return;
        isSubmittingVormerken = true;
        try {
            const res = await apiFetch("/api/vormerkungen", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ titel_id: titelId, schueler_id: activeStudent.id, notiz: "Vorgemerkt im Kiosk" })
            });
            if (!res.ok) throw new Error(await res.text());
            triggerFlash("success", "Erfolgreich vorgemerkt!");
            showVormerkenModal = false;
            vormerkenQuery = "";
            vormerkenResults = [];
        } catch(e) {
            triggerFlash("error", "Fehler beim Vormerken");
        } finally {
            isSubmittingVormerken = false;
        }
    }

    async function handleChecklistSubmit() {
        if (!pendingGeraet || !activeStudent || checklistItems.length !== checkedItems.size) return;
        isSubmittingChecklist = true;
        try {
            const res = await apiFetch("/api/action", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ 
                    query: pendingGeraetQuery, 
                    active_student_id: activeStudent.id,
                    confirmed_checklist: true
                })
            });
            if (!res.ok) throw new Error(await res.text());
            const data = await res.json();
            
            if (data.type === "ausleihe") {
                scannedBooks = [data.geraet, ...scannedBooks];
                triggerFlash("success");
            } else if (data.type === "rueckgabe") {
                returnedBook = data.geraet;
                returnedLoanId = data.loan_id || (data.loanID ? data.loanID : "");
                showDamageInput = false;
                damageDescription = "";
                playSuccessBeep();
            }
            showChecklistModal = false;
            pendingGeraet = null;
        } catch (e) {
            triggerFlash("error", e instanceof Error ? e.message : "Fehler beim Buchen des Geräts.");
        } finally {
            isSubmittingChecklist = false;
            focusBookInput();
        }
    }

    return {
        // Getter
        get activeStudent() { return activeStudent; },
        get studentInputVal() { return studentInputVal; },
        set studentInputVal(v) { studentInputVal = v; },
        get bookInputVal() { return bookInputVal; },
        set bookInputVal(v) { bookInputVal = v; },
        get scannedBooks() { return scannedBooks; },
        get toast() { return toast; },
        get screenFlash() { return screenFlash; },
        get isShaking() { return isShaking; },
        get isScanningStudent() { return isScanningStudent; },
        get isScanningBook() { return isScanningBook; },
        get returnedBook() { return returnedBook; },
        set returnedBook(v) { returnedBook = v; },
        get returnedLoanId() { return returnedLoanId; },
        get showDamageInput() { return showDamageInput; },
        set showDamageInput(v) { showDamageInput = v; },
        get damageDescription() { return damageDescription; },
        set damageDescription(v) { damageDescription = v; },
        get isSubmittingDamage() { return isSubmittingDamage; },
        get showVormerkenModal() { return showVormerkenModal; },
        set showVormerkenModal(v) { showVormerkenModal = v; },
        get vormerkenQuery() { return vormerkenQuery; },
        set vormerkenQuery(v) { vormerkenQuery = v; },
        get vormerkenResults() { return vormerkenResults; },
        get isSearchingVormerken() { return isSearchingVormerken; },
        get isSubmittingVormerken() { return isSubmittingVormerken; },
        get showChecklistModal() { return showChecklistModal; },
        set showChecklistModal(v) { showChecklistModal = v; },
        get pendingGeraet() { return pendingGeraet; },
        set pendingGeraet(v) { pendingGeraet = v; },
        get checklistItems() { return checklistItems; },
        get checkedItems() { return checkedItems; },
        set checkedItems(v) { checkedItems = v; },
        get isSubmittingChecklist() { return isSubmittingChecklist; },
        get systemSettings() { return systemSettings; },
        get isLimitReached() { return isLimitReached; },
        get isStudentBlocked() { return isStudentBlocked; },
        
        // Methoden
        triggerFlash,
        clearSession,
        focusStudentInput,
        focusBookInput,
        fetchSettings,
        handleStudentSubmit,
        handleBookSubmit,
        handleDamageOk,
        handleDamageSubmit,
        handleVormerkenSearch,
        handleVormerkenSubmit,
        handleChecklistSubmit
    };
}

export const kioskStore = createKioskStore();
