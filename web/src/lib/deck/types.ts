export type Figure = { label: string; value: string; note?: string; tone?: 'alert' };
export type Quote = { text: string; who: string };
/** One company's figure a year apart, drawn as a pair of bars. */
export type Bar = { label: string; before: number; after: number };
export type Column = { heading: string; items: string[] };

/**
 * A slide's layout is its kind; each kind reads only the fields it needs.
 *
 * - cover: the opening and closing slides. `record` adds the example sale.
 * - figures: up to three large numbers.
 * - bars: before and after bars for a few companies, in naira billions.
 * - chain: the supply chain as connected tiers, with figures beneath.
 * - compare: how it works today against how it works on Kredit.
 * - split: headline on the left, numbered points on the right.
 * - steps: a sequence laid out left to right.
 * - quote: one sourced quotation.
 */
export type Slide = {
	kind: 'cover' | 'figures' | 'bars' | 'chain' | 'compare' | 'split' | 'steps' | 'quote';
	eyebrow?: string;
	title: string;
	lede?: string;
	meta?: string;
	record?: boolean;
	figures?: Figure[];
	bars?: Bar[];
	barLabels?: [string, string];
	tiers?: [string, string][];
	compare?: [Column, Column];
	points?: [string, string][];
	quote?: Quote;
	note?: string;
	links?: [string, string][];
	source?: string;
};
