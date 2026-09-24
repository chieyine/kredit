declare global {
	namespace App {
		interface Locals {
			requestId?: string;
		}
		// Fields the root layout reads from any page's data to build its
		// <head>: a guide's article metadata, a page's own SEO copy, and whether
		// a legal document is currently published.
		interface PageData {
			article?: {
				title: string;
				description: string;
				published?: string;
				modified?: string;
				wordCount?: number;
				category?: string;
			};
			seo?: import('$lib/seo').PageSEO;
			legal?: { active?: boolean };
		}
	}
}

export {};
