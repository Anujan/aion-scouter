<script lang="ts">
	import type { Source } from '$lib/api';
	import { fmtCompact, fmtFrac, fmtInt, fmtOne, kindLabel } from '$lib/format';

	let { sources }: { sources: Source[] } = $props();

	const dealing = $derived(sources.filter((s) => s.damage > 0));
	const top = $derived(dealing[0]?.share ?? 1);
	const byKind = $derived.by(() => {
		const totals: Record<string, number> = {};
		for (const s of dealing) totals[s.kind] = (totals[s.kind] ?? 0) + s.share;
		return Object.entries(totals).sort((a, b) => b[1] - a[1]);
	});
</script>

<div class="kinds" role="img" aria-label="Damage share by kind">
	{#each byKind as [kind, share] (kind)}
		<span class="seg" style="flex: {share}; background: var(--kind-{kind})" title="{kindLabel[kind]} {fmtFrac(share)}"></span>
	{/each}
</div>
<div class="legend">
	{#each byKind as [kind, share] (kind)}
		<span><i style="background: var(--kind-{kind})"></i>{kindLabel[kind]} <b class="num">{fmtFrac(share)}</b></span>
	{/each}
</div>

<table>
	<thead>
		<tr>
			<th>Source</th>
			<th class="bar-col">Share</th>
			<th class="r">DPS</th>
			<th class="r hide-sm">Casts</th>
			<th class="r hide-sm">Hits</th>
			<th class="r hide-sm">Avg hit</th>
		</tr>
	</thead>
	<tbody>
		{#each dealing as s (s.name)}
			<tr>
				<td>
					<span class="dot" style="background: var(--kind-{s.kind})"></span>{s.name}
				</td>
				<td class="bar-col">
					<div class="bar">
						<span style="width: {(s.share / top) * 100}%; background: var(--kind-{s.kind})"></span>
					</div>
					<span class="num pct">{fmtFrac(s.share)}</span>
				</td>
				<td class="r num">{fmtInt(s.dps)}</td>
				<td class="r num hide-sm">{s.casts ? fmtOne(s.casts) : '–'}</td>
				<td class="r num hide-sm">{fmtOne(s.hits)}</td>
				<td class="r num hide-sm">{fmtCompact(s.avgHit)}</td>
			</tr>
		{/each}
	</tbody>
</table>

<style>
	.kinds {
		display: flex;
		height: 10px;
		border-radius: 999px;
		overflow: hidden;
		gap: 2px;
		margin-bottom: 0.5rem;
	}
	.legend {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem 1rem;
		font-size: 0.85rem;
		color: var(--muted);
		margin-bottom: 0.75rem;
	}
	.legend i {
		display: inline-block;
		width: 9px;
		height: 9px;
		border-radius: 2px;
		margin-right: 0.35rem;
	}
	.legend b {
		color: var(--text);
		font-weight: 600;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.88rem;
	}
	th {
		text-align: left;
		font-weight: 600;
		color: var(--muted);
		font-size: 0.78rem;
		padding: 0.3rem 0.4rem;
		border-bottom: 1px solid var(--border);
	}
	td {
		padding: 0.32rem 0.4rem;
		border-bottom: 1px solid var(--surface-2);
	}
	.r {
		text-align: right;
	}
	.dot {
		display: inline-block;
		width: 8px;
		height: 8px;
		border-radius: 50%;
		margin-right: 0.5rem;
		vertical-align: middle;
	}
	.bar-col {
		width: 32%;
		white-space: nowrap;
	}
	.bar {
		display: inline-block;
		width: calc(100% - 3.6rem);
		height: 8px;
		background: var(--surface-2);
		border-radius: 4px;
		vertical-align: middle;
		overflow: hidden;
	}
	.bar span {
		display: block;
		height: 100%;
		border-radius: 4px;
	}
	.pct {
		display: inline-block;
		width: 3.3rem;
		text-align: right;
		color: var(--muted);
	}
	@media (max-width: 640px) {
		.hide-sm {
			display: none;
		}
		.bar-col {
			width: 40%;
		}
	}
</style>
