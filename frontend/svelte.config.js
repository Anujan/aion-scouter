import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		// A static single-page app: the Go server (or any static host) serves
		// index.html for every route, and all data comes from /api.
		adapter: adapter({ fallback: 'index.html' })
	}
};

export default config;
