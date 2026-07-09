export const toastStore = new class {
  /** @type {{id: number, message: string, type: 'success' | 'error' | 'info'}[]} */
  toasts = $state([]);

  #counter = 0;
  /** @type {Map<number, ReturnType<typeof setTimeout>>} */
  #timers = new Map();

  /**
   * @param {string} message
   * @param {'success' | 'error' | 'info'} [type='info']
   */
  addToast(message, type = 'info') {
    const id = this.#counter++;
    this.toasts.push({ id, message, type });
    this.#timers.set(id, setTimeout(() => this.removeToast(id), 5000));
  }

  /**
   * @param {number} id
   */
  removeToast(id) {
    const timer = this.#timers.get(id);
    if (timer !== undefined) {
      clearTimeout(timer);
      this.#timers.delete(id);
    }
    this.toasts = this.toasts.filter(t => t.id !== id);
  }
}();
