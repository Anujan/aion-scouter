import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		// In development the Go API runs separately; proxy /api to it.
		proxy: { '/api': process.env.API_URL ?? 'http://localhost:8080' }
	}
});
