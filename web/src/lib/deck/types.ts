export type Figure = { label: string; value: string; note?: string; tone?: 'alert' };
export type Rows = { columns: string[]; data: string[][] };
export type Quote = { text: string; who: string };

export type Slide = {
	kind: 'cover' | 'statement' | 'data' | 'steps';
	eyebrow?: string;
	title: string;
	lede?: string;
	meta?: string;
	figures?: Figure[];
	rows?: Rows;
	points?: [string, string][];
	quote?: Quote;
	note?: string;
	links?: [string, string][];
	source?: string;
};
