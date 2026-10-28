<script lang="ts">
	import { goto } from '$app/navigation';
	import { characterPath, getMeta, search, type Meta, type SearchRow } from '$lib/api';
	import { onMount } from 'svelte';

	let meta = $state<Meta>();
	let region = $state('eu');
	let name = $state('');
	let results = $state<SearchRow[]>([]);
	let searched = $state(false);
	let loading = $state(false);
	let error = $state('');
	let demo = $state<SearchRow[]>([]);

	onMount(async () => {
		try {
			meta = await getMeta();
			demo = await search('demo', '');
		} catch (e) {
			error = (e as Error).message;
		}
	});

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (name.trim().length < 2) return;
		loading = true;
		error = '';
		try {
			results = await search(region, name.trim());
			searched = true;
		} catch (e) {
			error = (e as Error).message;
		} finally {
			loading = false;
		}
	}

	const open = (r: SearchRow) => goto(characterPath(r.region, r.serverId, r.characterId));
</script>

<section class="hero">
	<h1>How much damage do you really do?</h1>
	<p class="muted">
		Look up an AION 2 character to estimate its DPS from its real gear, Daevanion boards and skills,
		and see which stat to raise next. Spiritmaster and Chanter are supported.
	</p>

	<form class="search card" onsubmit={submit}>
		<select bind:value={region} aria-label="Region">
			{#each meta?.regions.filter((r) => r.id !== 'demo') ?? [{ id: 'eu', label: 'Europe' }] as r (r.id)}
				<option value={r.id}>{r.label}</option>
			{/each}
		</select>
		<input bind:value={name} placeholder="Character name" aria-label="Character name" minlength="2" />
		<button class="primary" disabled={loading || name.trim().length < 2}>
			{loading ? 'Searching…' : 'Search'}
		</button>
	</form>
	{#if error}<p class="error">{error}</p>{/if}
</section>

{#if searched}
	<section>
		<div class="section-title">{results.length} result{results.length === 1 ? '' : 's'}</div>
		{#if results.length === 0}
			<p class="muted">No characters found with that name in this region.</p>
		{:else}
			<ul class="results">
				{#each results as r (r.region + r.serverId + r.characterId)}
					<li>
						<button class="row card" onclick={() => open(r)}>
							<span class="name">{r.name}</span>
							<span class="muted">Lv {r.level} {r.className || 'Unknown class'}</span>
							<span class="muted">{r.serverName}</span>
							{#if r.supported}
								<span class="badge ok">DPS available</span>
							{:else}
								<span class="badge">Profile only</span>
							{/if}
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
{/if}

{#if demo.length}
	<section class="demo">
		<div class="section-title">Or try a demo character</div>
		<ul class="results">
			{#each demo as r (r.characterId)}
				<li>
					<button class="row card" onclick={() => open(r)}>
						<span class="name">{r.name}</span>
						<span class="muted">Lv {r.level} {r.className}</span>
						<span class="badge">Demo</span>
					</button>
				</li>
			{/each}
		</ul>
	</section>
{/if}

<style>
	.hero {
		max-width: 720px;
		margin: 1rem 0 2rem;
	}
	h1 {
		font-size: clamp(1.6rem, 4vw, 2.3rem);
		letter-spacing: -0.02em;
		margin-bottom: 0.5rem;
	}
	.search {
		display: flex;
		gap: 0.5rem;
		margin-top: 1.25rem;
		padding: 0.6rem;
		flex-wrap: wrap;
	}
	.search input {
		flex: 1 1 200px;
	}
	.results {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		gap: 0.5rem;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 340px), 1fr));
	}
	.row {
		width: 100%;
		display: flex;
		align-items: center;
		gap: 0.75rem;
		text-align: left;
		flex-wrap: wrap;
	}
	.row:hover {
		border-color: var(--accent);
	}
	.name {
		font-weight: 650;
		margin-right: auto;
	}
	.demo {
		margin-top: 2rem;
		margin-bottom: 2rem;
	}
</style>
