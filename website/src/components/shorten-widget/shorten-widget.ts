import { query } from '@/lib/dom';

const COPY_RESET_MS = 4000;
const FLASH_DURATION_MS = 700;
const MS_PER_DAY = 86_400_000;
const QR_FALLBACK_SIZE = 160;
const HTTP_TOO_MANY_REQUESTS = 429;
const HTTP_UNPROCESSABLE_ENTITY = 422;
const API_BASE = import.meta.env.PUBLIC_API_BASE ?? 'https://goshort.app';

interface ShortenResponse {
	short_url: string;
	short_code: string;
	original_url: string;
	expires_at: string | null;
}

interface ErrorResponse {
	error?: { message?: string };
}

const form = query<HTMLFormElement>('[data-shorten-form]');
const urlInput = query<HTMLInputElement>('[data-url-input]');
const honeypot = query<HTMLInputElement>('[data-honeypot]');
const submitBtn = query<HTMLButtonElement>('[data-submit-btn]');
const resultEl = query<HTMLDivElement>('[data-shorten-result]');
const shortUrlEl = query<HTMLAnchorElement>('[data-short-url]');
const originalUrlEl = query<HTMLParagraphElement>('[data-original-url]');
const visitLink = query<HTMLAnchorElement>('[data-visit-link]');
const copyIconBtn = query<HTMLButtonElement>('[data-copy-icon-btn]');
const copyBtn = query<HTMLButtonElement>('[data-copy-btn]');
const copyLabel = query<HTMLSpanElement>('[data-copy-label]');
const qrBtn = query<HTMLButtonElement>('[data-qr-btn]');
const qrSection = query<HTMLDivElement>('[data-qr-section]');
const qrImg = query<HTMLImageElement>('[data-qr-img]');
const downloadPngBtn = query<HTMLButtonElement>('[data-download-png-btn]');
const downloadJpegBtn = query<HTMLButtonElement>('[data-download-jpeg-btn]');
const downloadSvgBtn = query<HTMLButtonElement>('[data-download-svg-btn]');
const copyQrBtn = query<HTMLButtonElement>('[data-copy-qr-btn]');
const expiresLabel = query<HTMLSpanElement>('[data-expires-label]');
const shortenAnotherBtn = query<HTMLButtonElement>('[data-shorten-another]');
const errorEl = query<HTMLParagraphElement>('[data-shorten-error]');

let currentCode = '';

function setLoading(loading: boolean): void {
	if (!submitBtn) return;
	submitBtn.disabled = loading;
	submitBtn.textContent = loading ? 'Shortening…' : 'Shorten';
}

function showError(msg: string): void {
	if (!errorEl) return;
	errorEl.textContent = msg;
	errorEl.style.opacity = '1';
	resultEl?.classList.add('hidden');
}

function hideError(): void {
	if (!errorEl) return;
	errorEl.style.opacity = '0';
}

function showResult(data: ShortenResponse): void {
	if (!resultEl || !shortUrlEl || !originalUrlEl) return;
	shortUrlEl.textContent = data.short_url;
	shortUrlEl.href = data.short_url;
	if (visitLink) visitLink.href = data.short_url;
	originalUrlEl.textContent = data.original_url;
	if (expiresLabel) {
		if (data.expires_at) {
			const days = Math.ceil(
				(new Date(data.expires_at).getTime() - Date.now()) / MS_PER_DAY,
			);
			expiresLabel.textContent = `· expires in ${days} day${days === 1 ? '' : 's'}`;
		} else {
			expiresLabel.textContent = '';
		}
	}
	// Reset QR state
	currentCode = data.short_code;
	qrSection?.removeAttribute('data-visible');
	qrBtn?.removeAttribute('data-active');
	qrBtn?.removeAttribute('data-loaded');
	if (qrImg) qrImg.src = '';
	resultEl.classList.remove('hidden');
	resultEl.setAttribute('data-flash', '');
	setTimeout(() => resultEl?.removeAttribute('data-flash'), FLASH_DURATION_MS);
}

function validateURL(raw: string): string | null {
	const trimmed = raw.trim();
	if (!trimmed) return 'Please enter a URL.';
	try {
		const url = new URL(trimmed);
		if (url.protocol !== 'http:' && url.protocol !== 'https:')
			return 'URL must start with http:// or https://';
		return null;
	} catch {
		return "That doesn't look like a valid URL.";
	}
}

async function copyShortUrl(): Promise<void> {
	const url = shortUrlEl?.textContent ?? '';
	if (url) await navigator.clipboard.writeText(url);
}

form?.addEventListener('submit', async (e) => {
	e.preventDefault();
	hideError();

	const raw = urlInput?.value ?? '';
	const validationError = validateURL(raw);
	if (validationError) {
		showError(validationError);
		return;
	}

	setLoading(true);

	try {
		const res = await fetch(`${API_BASE}/api/v1/urls/public`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				url: raw.trim(),
				website: honeypot?.value ?? '',
			}),
		});

		if (res.ok) {
			const data = (await res.json()) as ShortenResponse;
			showResult(data);
		} else if (res.status === HTTP_TOO_MANY_REQUESTS) {
			showError('Too many requests — please wait a moment and try again.');
		} else if (res.status === HTTP_UNPROCESSABLE_ENTITY) {
			showError('URL flagged as potentially unsafe.');
		} else {
			const body = (await res.json().catch(() => null)) as ErrorResponse | null;
			showError(
				body?.error?.message ?? 'Invalid URL. Please check and try again.',
			);
		}
	} catch {
		showError('Network error. Please check your connection and try again.');
	} finally {
		setLoading(false);
	}
});

copyIconBtn?.addEventListener('click', async () => {
	if (!copyIconBtn) return;
	await copyShortUrl();
	copyIconBtn.dataset.copied = '';
	setTimeout(() => {
		delete copyIconBtn.dataset.copied;
	}, COPY_RESET_MS);
});

copyBtn?.addEventListener('click', async () => {
	if (!copyBtn) return;
	await copyShortUrl();
	copyBtn.dataset.copied = '';
	if (copyLabel) copyLabel.textContent = 'Copied!';
	setTimeout(() => {
		if (!copyBtn) return;
		delete copyBtn.dataset.copied;
		if (copyLabel) copyLabel.textContent = 'Copy';
	}, COPY_RESET_MS);
});

qrBtn?.addEventListener('click', () => {
	if (!qrSection || !qrImg || !currentCode) return;
	// Lazy-load QR image on first reveal
	if (!qrBtn.hasAttribute('data-loaded')) {
		qrImg.src = `${API_BASE}/api/v1/urls/${currentCode}/qr?size=160`;
		qrBtn.setAttribute('data-loaded', '');
	}
	const nowVisible = !qrSection.hasAttribute('data-visible');
	qrSection.toggleAttribute('data-visible', nowVisible);
	qrBtn.toggleAttribute('data-active', nowVisible);
});

// Blob fetch for cross-origin download (goshort.ngockhoi96.dev ≠ goshort.app)
async function downloadQR(
	format: 'png' | 'jpeg' | 'svg',
	filename: string,
): Promise<void> {
	if (!currentCode) return;
	try {
		const res = await fetch(
			`${API_BASE}/api/v1/urls/${currentCode}/qr?format=${format}`,
		);
		const blob = await res.blob();
		const objectUrl = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = objectUrl;
		a.download = filename;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		URL.revokeObjectURL(objectUrl);
	} catch {
		// silently fail — user can right-click the QR image and save manually
	}
}

downloadPngBtn?.addEventListener('click', () =>
	downloadQR('png', `qr-${currentCode}.png`),
);
downloadJpegBtn?.addEventListener('click', () =>
	downloadQR('jpeg', `qr-${currentCode}.jpg`),
);
downloadSvgBtn?.addEventListener('click', () =>
	downloadQR('svg', `qr-${currentCode}.svg`),
);

// Draw already-loaded QR img onto canvas → toBlob → clipboard (no extra fetch).
// crossOrigin="anonymous" on the img + CORS on the QR endpoint prevents canvas taint.
copyQrBtn?.addEventListener('click', async () => {
	if (!qrImg?.src || !copyQrBtn) return;
	try {
		const canvas = document.createElement('canvas');
		canvas.width = qrImg.naturalWidth || QR_FALLBACK_SIZE;
		canvas.height = qrImg.naturalHeight || QR_FALLBACK_SIZE;
		const ctx = canvas.getContext('2d');
		if (!ctx) return;
		ctx.drawImage(qrImg, 0, 0);
		const blob = await new Promise<Blob | null>((resolve) =>
			canvas.toBlob(resolve, 'image/png'),
		);
		if (!blob) return;
		await navigator.clipboard.write([new ClipboardItem({ 'image/png': blob })]);
		copyQrBtn.dataset.copied = '';
		setTimeout(() => {
			if (!copyQrBtn) return;
			delete copyQrBtn.dataset.copied;
		}, COPY_RESET_MS);
	} catch {
		// silently fail — canvas tainted or ClipboardItem not supported
	}
});

urlInput?.addEventListener('input', hideError);

shortenAnotherBtn?.addEventListener('click', () => {
	if (!urlInput || !resultEl) return;
	resultEl.classList.add('hidden');
	urlInput.value = '';
	hideError();
	urlInput.focus();
});
