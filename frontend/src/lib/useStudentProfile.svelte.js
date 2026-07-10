import { apiFetch, apiClient } from "./apiFetch.js";

export function useStudentProfile() {
    /** @type {any} */
    let profile = $state(null);
    /** @type {any[]} */
    let vormerkungen = $state([]);
    let loading = $state(true);
    let showWebcam = $state(false);
    let timestamp = $state(Date.now());
    let showDeleteConfirm = $state(false);
    let activeTab = $state("ausleihen");
    let showEditModal = $state(false);
    let showDamageModal = $state(false);
    let showLockModal = $state(false);
    let damageBook = $state(null);
    let isSubmittingDamage = $state(false);
    let globalErrorToast = $state(null);
    let rechnungPdfLoading = $state(false);
    let kontoauszugPdfLoading = $state(false);

    async function fetchProfile(studentId) {
        if (!studentId) return;
        loading = true;
        try {
            const [resProfile, resVormerkungen] = await Promise.all([
                apiFetch(`/api/schueler/${studentId}`),
                apiFetch(`/api/vormerkungen?schueler_id=${studentId}`)
            ]);
            if (resProfile.ok) profile = await resProfile.json();
            if (resVormerkungen.ok) vormerkungen = await resVormerkungen.json();
        } catch (err) {
            console.error("Fehler beim Laden des Schüler-Profils:", err);
        } finally {
            loading = false;
        }
    }

    function handleDeleteSuccess(onDeselect) {
        showDeleteConfirm = false;
        if (onDeselect) onDeselect();
    }

    function handleSaveEdit(studentId) {
        showEditModal = false;
        fetchProfile(studentId);
    }

    function handlePhotoCaptured(studentId) {
        timestamp = Date.now();
        showWebcam = false;
        fetchProfile(studentId);
    }

    async function downloadRechnungPDF() {
        if (!profile) return;
        rechnungPdfLoading = true;
        globalErrorToast = null;
        try {
            const res = await apiFetch(`/api/print/rechnung/${profile.id}`);
            if (!res.ok) {
                const errText = await res.text();
                throw new Error(errText || "Keine ausstehenden Rechnungen gefunden");
            }
            const blob = await res.blob();
            const url = URL.createObjectURL(blob);
            const a = document.createElement("a");
            a.href = url;
            a.download = `Rechnung_${profile.vorname}_${profile.nachname}.pdf`;
            a.click();
            URL.revokeObjectURL(url);
        } catch (e) {
            globalErrorToast = String(e);
            setTimeout(() => globalErrorToast = null, 4000);
        } finally {
            rechnungPdfLoading = false;
        }
    }

    async function downloadKontoauszugPDF() {
        if (!profile) return;
        kontoauszugPdfLoading = true;
        globalErrorToast = null;
        try {
            const res = await apiFetch(`/api/print/kontoauszug/${profile.id}`);
            if (!res.ok) {
                const errText = await res.text();
                throw new Error(errText || "Keine aktiven Ausleihen gefunden");
            }
            const blob = await res.blob();
            const url = URL.createObjectURL(blob);
            const a = document.createElement("a");
            a.href = url;
            a.download = `Kontoauszug_${profile.vorname}_${profile.nachname}.pdf`;
            a.click();
            URL.revokeObjectURL(url);
        } catch (e) {
            globalErrorToast = String(e);
            setTimeout(() => globalErrorToast = null, 4000);
        } finally {
            kontoauszugPdfLoading = false;
        }
    }

    function openDamageModal(book) {
        damageBook = book;
        showDamageModal = true;
    }

    async function submitDamageReport(studentId, reason, amount) {
        if (!damageBook) return;
        isSubmittingDamage = true;
        try {
            const res = await apiClient.post(`/api/damage/report`, {
                loan_id: damageBook.ausleihe_id,
                schueler_id: studentId,
                // BorrowedBook liefert die Exemplar-ID als `id` — ein
                // `exemplar_id`-Feld gibt es dort nicht (500: leere UUID).
                copy_id: damageBook.id,
                beschreibung: reason,
                betrag: amount
            });
            if (res.ok) {
                const json = await res.json();
                window.open(`/api/schadensfaelle/${json.schadens_id}/pdf`, '_blank');
                showDamageModal = false;
                fetchProfile(studentId);
            } else {
                const err = await res.json().catch(() => ({}));
                alert(err.error || "Fehler beim Melden.");
            }
        } catch (e) {
            alert("Netzwerkfehler.");
        } finally {
            isSubmittingDamage = false;
        }
    }

    function handleLockSuccess(updated) {
        if (profile) {
            profile.is_manually_blocked = updated.is_manually_blocked;
            profile.ist_gesperrt = profile.is_manually_blocked || profile.has_open_damages;
        }
    }

    return {
        get profile() { return profile; },
        set profile(v) { profile = v; },
        get vormerkungen() { return vormerkungen; },
        set vormerkungen(v) { vormerkungen = v; },
        get loading() { return loading; },
        get showWebcam() { return showWebcam; },
        set showWebcam(v) { showWebcam = v; },
        get timestamp() { return timestamp; },
        get showDeleteConfirm() { return showDeleteConfirm; },
        set showDeleteConfirm(v) { showDeleteConfirm = v; },
        get activeTab() { return activeTab; },
        set activeTab(v) { activeTab = v; },
        get showEditModal() { return showEditModal; },
        set showEditModal(v) { showEditModal = v; },
        get showDamageModal() { return showDamageModal; },
        set showDamageModal(v) { showDamageModal = v; },
        get showLockModal() { return showLockModal; },
        set showLockModal(v) { showLockModal = v; },
        get damageBook() { return damageBook; },
        get isSubmittingDamage() { return isSubmittingDamage; },
        get globalErrorToast() { return globalErrorToast; },
        get rechnungPdfLoading() { return rechnungPdfLoading; },
        get kontoauszugPdfLoading() { return kontoauszugPdfLoading; },
        fetchProfile,
        handleDeleteSuccess,
        handleSaveEdit,
        handlePhotoCaptured,
        downloadRechnungPDF,
        downloadKontoauszugPDF,
        openDamageModal,
        submitDamageReport,
        handleLockSuccess
    };
}
