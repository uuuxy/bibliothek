// Svelte 5 reactive array state for dynamic extensions
export const sidebarExtensions = $state([]);
export const studentTabExtensions = $state([]);

/**
 * Registers a component to be rendered in the main sidebar.
 * @param {any} component - Svelte component to render
 * @param {any} [props] - Optional props object to pass to the component
 */
export function registerSidebarExtension(component, props = {}) {
	sidebarExtensions.push({ component, props });
}

/**
 * Registers a component to be rendered as an extra section in the student profile tab.
 * @param {string} name - User-friendly label for this extension area
 * @param {any} component - Svelte component to render
 * @param {any} [props] - Optional props object to pass to the component
 */
export function registerStudentTabExtension(name, component, props = {}) {
	studentTabExtensions.push({ name, component, props });
}
