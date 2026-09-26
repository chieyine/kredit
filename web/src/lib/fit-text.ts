/**
 * Keeps a large figure on one line. ₦1,876,588,210.99 on a small phone would
 * otherwise break in the middle of the number, which reads as two amounts.
 * The text starts at its designed size and steps down only as far as needed.
 */
export function fitText(node: HTMLElement) {
	const fit = () => {
		node.style.fontSize = '';
		let size = parseFloat(getComputedStyle(node).fontSize);
		while (node.scrollWidth > node.clientWidth + 1 && size > 18) {
			size -= 1;
			node.style.fontSize = `${size}px`;
		}
	};
	fit();
	const resize = new ResizeObserver(fit);
	resize.observe(node);
	const changes = new MutationObserver(fit);
	changes.observe(node, { subtree: true, characterData: true, childList: true });
	return {
		destroy() {
			resize.disconnect();
			changes.disconnect();
		}
	};
}
