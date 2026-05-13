/**
 * Typed DOM query helpers.
 *
 * Use data-* attribute selectors to decouple JS from styling classes.
 * Example: queryData<HTMLButtonElement>('shorten-btn')
 */

export function query<T extends HTMLElement>(
	selector: string,
	parent: ParentNode = document,
): T | null {
	return parent.querySelector<T>(selector);
}

export function queryAll<T extends HTMLElement>(
	selector: string,
	parent: ParentNode = document,
): NodeListOf<T> {
	return parent.querySelectorAll<T>(selector);
}

export function queryData<T extends HTMLElement>(
	name: string,
	value?: string,
	parent: ParentNode = document,
): T | null {
	const selector = value != null ? `[data-${name}="${value}"]` : `[data-${name}]`;
	return parent.querySelector<T>(selector);
}

export function queryAllData<T extends HTMLElement>(
	name: string,
	value?: string,
	parent: ParentNode = document,
): NodeListOf<T> {
	const selector = value != null ? `[data-${name}="${value}"]` : `[data-${name}]`;
	return parent.querySelectorAll<T>(selector);
}
