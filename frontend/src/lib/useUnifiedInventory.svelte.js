import { apiFetch } from './apiFetch.js';

/**
 * Hook for Unified Inventory state and logic
 */
export function useUnifiedInventory() {
    let status = $state('idle'); // 'idle', 'active'
    let scopeType = $state('global');
    let selectedSignatureId = $state('');
    let signatures = $state(/** @type {any[]} */ ([]));
    let stats = $state({ erwartet: 0, erfasst: 0 });
    let lastScan = $state(/** @type {any} */ (null));
    let barcodeInput = $state('');
    let isScanning = $state(false);
    let showStartModal = $state(false);
    let showFinishModal = $state(false);
    let errorMessage = $state('');

    async function loadSignatures() {
        try {
            const res = await apiFetch('/api/signatures');
            if (res.ok) {
                signatures = await res.json();
            }
        } catch (e) {
            console.error("Failed to load signatures", e);
        }
    }

    async function startInventory() {
        errorMessage = '';
        const payload = { type: scopeType };
        if (scopeType === 'signature') {
            if (!selectedSignatureId) {
                errorMessage = 'Bitte wähle eine Signatur aus.';
                return;
            }
            // @ts-ignore
            payload.signature_id = Number(selectedSignatureId);
        }

        try {
            const res = await apiFetch('/api/inventur/start', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            if (res.ok) {
                const data = await res.json();
                stats.erwartet = data.erwartet;
                stats.erfasst = 0;
                lastScan = null;
                status = 'active';
                showStartModal = false;
            } else {
                errorMessage = 'Fehler beim Starten der Inventur.';
            }
        } catch (e) {
            errorMessage = 'Netzwerkfehler.';
        }
    }

    /** @param {Function} focusInput */
    async function handleScan(barcodeVal, focusInput) {
        if (!barcodeVal.trim() || isScanning) return;
        
        isScanning = true;
        const barcode = barcodeVal.trim();
        barcodeInput = '';
        
        try {
            const res = await apiFetch('/api/inventur/scan', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ barcode_id: barcode })
            });
            
            if (res.ok) {
                const data = await res.json();
                stats.erfasst++;
                lastScan = {
                    success: true,
                    barcode: data.barcode_id,
                    title: data.titel,
                    warnings: data.warnungen || []
                };
            } else {
                const err = await res.json().catch(() => ({}));
                lastScan = {
                    success: false,
                    barcode: barcode,
                    title: 'Unbekanntes Buch',
                    warnings: [err.error || 'Scan fehlgeschlagen']
                };
            }
        } catch (e) {
            lastScan = {
                success: false,
                barcode: barcode,
                title: 'Fehler',
                warnings: ['Netzwerkfehler beim Scannen']
            };
        } finally {
            isScanning = false;
            if (focusInput) focusInput();
        }
    }

    async function finishInventory() {
        try {
            const res = await apiFetch('/api/inventur/finish', { method: 'POST' });
            if (res.ok) {
                const data = await res.json();
                alert(`Inventur abgeschlossen! ${data.verloren_gemeldet} Bücher wurden als verloren markiert.`);
                status = 'idle';
                showFinishModal = false;
                stats = { erwartet: 0, erfasst: 0 };
                lastScan = null;
            } else {
                alert('Fehler beim Abschließen der Inventur.');
            }
        } catch (e) {
            alert('Netzwerkfehler.');
        }
    }

    function getProgressPercent() {
        if (stats.erwartet === 0) return 0;
        return Math.min(100, Math.round((stats.erfasst / stats.erwartet) * 100));
    }

    return {
        get status() { return status; },
        get scopeType() { return scopeType; },
        set scopeType(v) { scopeType = v; },
        get selectedSignatureId() { return selectedSignatureId; },
        set selectedSignatureId(v) { selectedSignatureId = v; },
        get signatures() { return signatures; },
        get stats() { return stats; },
        get lastScan() { return lastScan; },
        get barcodeInput() { return barcodeInput; },
        set barcodeInput(v) { barcodeInput = v; },
        get isScanning() { return isScanning; },
        get showStartModal() { return showStartModal; },
        set showStartModal(v) { showStartModal = v; },
        get showFinishModal() { return showFinishModal; },
        set showFinishModal(v) { showFinishModal = v; },
        get errorMessage() { return errorMessage; },
        loadSignatures,
        startInventory,
        handleScan,
        finishInventory,
        getProgressPercent
    };
}
