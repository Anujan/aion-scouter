<script lang="ts">
	import type { StatEntry } from '$lib/api';
	import { fmtStat } from '$lib/format';

	let { stats }: { stats: StatEntry[] } = $props();

	const groups = [
		['offense', 'Offense'],
		['utility', 'Utility'],
		['defense', 'Defense'],
		['pvp', 'PvP'],
		['other', 'Other']
	] as const;
	let open = $state<string | null>(null);
</script>

<p class="muted lead">
	NCSOFT doesn't publish the in-game stat window, so these totals are rebuilt from gear,
	manastones, set bonuses, titles, Daevanion nodes, passives and primary stats. Click a stat to see
	where it comes from. If a number differs from your stat window, enter the correct value under
	"Match your stat window" on the DPS tab.
</p>
<div class="groups">
	{#each groups as [id, title] (id)}
		{@const rows = stats.filter((s) => s.group === id)}
		{#if rows.length}
			<div class="card">
				<div class="section-title">{title}</div>
				<dl>
					{#each rows as s (s.key)}
						<button class="stat" class:open={open === s.key} onclick={() => (open = open === s.key ? null : s.key)}>
							<dt>{s.label}</dt>
							<dd class="num">{fmtStat(s.value, s.pct)}</dd>
						</button>
						{#if open === s.key}
							<ul class="sources">
								{#each s.sources ?? [] as c, i (i)}
									<li>
										<span class="muted">{c.source}</span>
										<span>{c.detail}</span>
										<span class="num">{c.value > 0 ? '+' : ''}{fmtStat(c.value, s.pct)}</span>
									</li>
								{/each}
							</ul>
						{/if}
					{/each}
				</dl>
			</div>
		{/if}
	{/each}
</div>

<style>
	.lead {
		font-size: 0.85rem;
		margin: 0 0 1rem;
		max-width: 75ch;
	}
	.groups {
		display: grid;
		gap: 0.75rem;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 320px), 1fr));
		align-items: start;
	}
	dl {
		margin: 0;
	}
	.stat {
		display: flex;
		justify-content: space-between;
		width: 100%;
		border: 0;
		border-radius: 6px;
		padding: 0.25rem 0.4rem;
		background: none;
		text-align: left;
	}
	.stat:hover,
	.stat.open {
		background: var(--surface-2);
	}
	dt,
	dd {
		margin: 0;
	}
	dd {
		font-weight: 600;
	}
	.sources {
		list-style: none;
		margin: 0.15rem 0 0.4rem;
		padding: 0.4rem 0.6rem;
		font-size: 0.8rem;
		border-left: 2px solid var(--accent);
	}
	.sources li {
		display: grid;
		grid-template-columns: 6.5rem 1fr auto;
		gap: 0.5rem;
	}
</style>
