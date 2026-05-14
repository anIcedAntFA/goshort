// @ts-check
import { fileURLToPath } from 'node:url';

import sitemap from '@astrojs/sitemap';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'astro/config';

export default defineConfig({
	site: 'https://goshort.ngockhoi96.dev',
	output: 'static',
	integrations: [sitemap()],
	vite: {
		plugins: [tailwindcss()],
		resolve: {
			alias: {
				'@': fileURLToPath(new URL('./src', import.meta.url)),
			},
		},
	},
});
