<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Aion Scouter</title>
</svelte:head>

<header>
	<div class="wrap">
		<a class="brand" href="/">Aion<span>Scouter</span></a>
		<span class="muted tag">AION 2 damage estimates</span>
	</div>
</header>

<main class="wrap">
	{@render children()}
</main>

<footer class="wrap muted">
	Unofficial fan project, not affiliated with NCSOFT. Character data comes from the official AION 2
	character site; skill and Daevanion values come from the Global client via metabot.gg. Numbers are
	estimates. Each result lists the assumptions behind it.
</footer>

<style>
	.wrap {
		max-width: 1180px;
		margin: 0 auto;
		padding: 0 16px;
	}
	header {
		border-bottom: 1px solid var(--border);
		background: var(--surface);
		margin-bottom: 1.5rem;
	}
	header .wrap {
		display: flex;
		align-items: baseline;
		gap: 0.9rem;
		padding-top: 0.9rem;
		padding-bottom: 0.9rem;
	}
	.brand {
		font-weight: 800;
		font-size: 1.25rem;
		color: var(--text);
		letter-spacing: -0.02em;
	}
	.brand span {
		color: var(--accent);
	}
	.brand:hover {
		text-decoration: none;
	}
	.tag {
		font-size: 0.85rem;
	}
	footer {
		font-size: 0.8rem;
		padding-top: 2.5rem;
		padding-bottom: 2rem;
	}
</style>
