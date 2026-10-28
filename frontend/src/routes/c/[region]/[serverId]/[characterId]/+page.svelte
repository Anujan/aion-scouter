<script lang="ts">
	import { page } from '$app/state';
	import { getCharacter, getMeta, type Character, type Meta } from '$lib/api';
	import DpsPanel from '$lib/components/DpsPanel.svelte';
	import GearGrid from '$lib/components/GearGrid.svelte';
	import SkillsBoards from '$lib/components/SkillsBoards.svelte';
	import StatSheet from '$lib/components/StatSheet.svelte';
	import { fmtInt } from '$lib/format';

	let character = $state<Character>();
	let meta = $state<Meta>();
	let error = $state('');
	let tab = $state<'dps' | 'stats' | 'gear' | 'skills'>('dps');

	$effect(() => {
		const { region, serverId, characterId } = page.params;
		character = undefined;
		error = '';
		Promise.all([getCharacter(region!, serverId!, characterId!), getMeta()])
			.then(([c, m]) => {
				character = c;
				meta = m;
				if (!c.class.supported) tab = 'stats';
			})
			.catch((e) => (error = e.message));
	});

	const tabs = [
		['dps', 'DPS'],
		['stats', 'Stats'],
		['gear', 'Gear'],
		['skills', 'Skills & Daevanion']
	] as const;
</script>

<svelte:head>
	<title>{character ? `${character.profile.name} · Aion Scouter` : 'Aion Scouter'}</title>
</svelte:head>

{#if error}
	<div class="card">
		<p class="error">{error}</p>
		<a href="/">Back to search</a>
	</div>
{:else if !character || !meta}
	<div class="card loading">
		<p>Loading character from the official site…</p>
		<p class="muted">The first load reads every equipped item and Daevanion board, which takes about 10 seconds.</p>
	</div>
{:else}
	{@const p = character.profile}
	<section class="profile">
		{#if p.profileImage}<img src={p.profileImage} alt="" width="64" height="64" />{/if}
		<div>
			<h1>{p.name}</h1>
			<div class="muted">
				{[`Lv ${p.level} ${p.className}`, p.server, p.race, p.title && `“${p.title}”`].filter(Boolean).join(' · ')}
			</div>
		</div>
		<div class="facts">
			<div><span class="muted">Combat Power</span><b class="num">{fmtInt(p.combatPower)}</b></div>
			<div><span class="muted">Item Level</span><b class="num">{fmtInt(p.itemLevel)}</b></div>
			{#if character.class.supported}
				<span class="badge warn" title="Estimates are calibrated against community formulas, not yet against DPS-meter logs">
					{character.class.status}
				</span>
			{/if}
		</div>
	</section>

	<nav class="tabs" aria-label="Character sections">
		{#each tabs as [id, label] (id)}
			<button
				class:active={tab === id}
				onclick={() => (tab = id)}
				disabled={id === 'dps' && !character.class.supported}
			>
				{label}
			</button>
		{/each}
	</nav>

	{#if tab === 'dps'}
		{#key character.ref.characterId}
			<DpsPanel {character} {meta} />
		{/key}
	{:else if tab === 'stats'}
		{#if !character.class.supported}
			<p class="muted">DPS estimates aren't available for {p.className} yet. Spiritmaster and Chanter are supported.</p>
		{/if}
		<StatSheet stats={character.stats} />
	{:else if tab === 'gear'}
		<GearGrid gear={character.equipment} />
	{:else}
		<SkillsBoards skills={character.skills} boards={character.boards} />
	{/if}
{/if}

<style>
	.loading {
		max-width: 560px;
	}
	.profile {
		display: flex;
		align-items: center;
		gap: 1rem;
		flex-wrap: wrap;
		margin-bottom: 1rem;
	}
	.profile img {
		border-radius: 12px;
		background: var(--surface-2);
	}
	h1 {
		font-size: 1.7rem;
		letter-spacing: -0.02em;
	}
	.facts {
		display: flex;
		gap: 1.25rem;
		align-items: center;
		margin-left: auto;
	}
	.facts div {
		display: flex;
		flex-direction: column;
		font-size: 0.78rem;
	}
	.facts b {
		font-size: 1.15rem;
	}
	.tabs {
		display: flex;
		gap: 0.25rem;
		border-bottom: 1px solid var(--border);
		margin-bottom: 1rem;
		overflow-x: auto;
	}
	.tabs button {
		border: 0;
		border-bottom: 2px solid transparent;
		border-radius: 0;
		background: none;
		padding: 0.55rem 0.9rem;
		color: var(--muted);
		font-weight: 600;
		white-space: nowrap;
	}
	.tabs button.active {
		color: var(--text);
		border-bottom-color: var(--accent);
	}
	@media (max-width: 640px) {
		.facts {
			margin-left: 0;
		}
	}
</style>
