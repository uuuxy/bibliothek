export function leseCsrfToken() {
    if (typeof document === "undefined") {
        return "";
    }

    const cookieEintraege = document.cookie ? document.cookie.split("; ") : [];
    for (const eintrag of cookieEintraege) {
        if (eintrag.startsWith("inventur_csrf=")) {
            const wert = eintrag.slice("inventur_csrf=".length);
            return decodeURIComponent(wert);
        }
    }

    return "";
}

/**
 * @returns {Record<string, string>}
 */
export function csrfHeader() {
    const token = leseCsrfToken();
    if (!token) {
        return {};
    }
    return { "X-CSRF-Token": token };
}
