export const toastStore = new class {
  /** @type {{id: number, message: string, type: 'success' | 'error' | 'info'}[]} */
  toasts = $state([]);
  
  #counter = 0;

  /**
   * @param {string} message 
   * @param {'success' | 'error' | 'info'} [type='info'] 
   */
  addToast(message, type = 'info') {
    const id = this.#counter++;
    this.toasts.push({ id, message, type });
    
    setTimeout(() => {
      this.removeToast(id);
    }, 4000);
  }

  /**
   * @param {number} id 
   */
  removeToast(id) {
    this.toasts = this.toasts.filter(t => t.id !== id);
  }
}();
