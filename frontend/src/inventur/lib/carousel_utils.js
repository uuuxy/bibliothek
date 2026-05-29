/**
 * @param {Event} event
 * @param {number} direction
 */
export function scrollCarousel(event, direction) {
    const target = /** @type {HTMLElement} */ (event.currentTarget);
    const container = target?.parentElement?.querySelector('.carousel-container');
    if (container) {
        const scrollAmount = container.clientWidth * 0.75;
        container.scrollBy({ left: direction * scrollAmount, behavior: 'smooth' });
    }
}

/**
 * @param {HTMLElement} node
 */
export function scrollHandler(node) {
    const checkScroll = () => {
        if (!node || !node.parentElement) return;
        const canScrollLeft = node.scrollLeft > 0;
        // Use a margin of 2px to prevent float rounding issues
        const canScrollRight = Math.ceil(node.scrollLeft + node.clientWidth) < node.scrollWidth - 2;

        node.parentElement.dataset.canScrollLeft = String(canScrollLeft);
        node.parentElement.dataset.canScrollRight = String(canScrollRight);
    };

    node.addEventListener('scroll', checkScroll, { passive: true });
    window.addEventListener('resize', checkScroll, { passive: true });

    // Initial checks need a small delay so images and DOM can layout
    setTimeout(checkScroll, 50);
    setTimeout(checkScroll, 500);

    return {
        destroy() {
            node.removeEventListener('scroll', checkScroll);
            window.removeEventListener('resize', checkScroll);
        }
    };
}