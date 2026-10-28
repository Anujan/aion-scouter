# Aion Scouter frontend

SvelteKit single-page app. It only displays data: every number comes from the Go API in `../backend`.

```sh
npm install
npm run dev     # http://localhost:5173, proxies /api to http://localhost:8080 (override with API_URL)
npm run check   # type-check
npm run build   # static build in ./build, served by `server -static`
```
