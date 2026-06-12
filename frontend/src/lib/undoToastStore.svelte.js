export const undoToasts = $state([]);

export function addUndoToast(message, onUndo) {
  const id = crypto.randomUUID();
  
  // Füge den neuen Toast am Anfang hinzu
  undoToasts.unshift({ id, message, onUndo });
  
  // Auto-Dismiss nach 5 Sekunden
  setTimeout(() => {
    removeUndoToast(id);
  }, 5000);
}

export function removeUndoToast(id) {
  const index = undoToasts.findIndex(t => t.id === id);
  if (index !== -1) {
    undoToasts.splice(index, 1);
  }
}
