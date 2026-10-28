<script lang="ts">
	import type { StatValue } from '$lib/api';
	import { fmtOne, fmtSigned } from '$lib/format';

	let { values }: { values: StatValue[] } = $props();

	const sorted = $derived(
		[...values].sort((a, b) => Number(a.withinNoise) - Number(b.withinNoise) || b.dpsGain - a.dpsGain)
	);
	const max = $derived(Math.max(...values.map((v) => Math.abs(v.attackEquivalent)), 1));
</script>

<p class="muted lead">
	Each row re-runs the simulation with one stat raised by the step shown. "Worth" converts the gain
	into flat Attack, so different stats can be compared on one scale. This is the same idea as
	MapleScouter's converted stat. Gains marked ≈ 0 are smaller than the simulation's
	run-to-run noise.
</p>
<table>
	<thead>
		<tr>
			<th>Stat</th>
			<th class="r">Step</th>
			<th class="r">DPS gain</th>
			<th class="bar-col">Worth (flat Attack)</th>
		</tr>
	</thead>
	<tbody>
		{#each sorted as v (v.key)}
			<tr>
				<td>{v.label}</td>
				<td class="r num muted">+{fmtOne(v.step)}{v.pct ? '%' : ''}</td>
				<td class="r num" class:zero={v.withinNoise} title={v.withinNoise ? 'Smaller than run-to-run noise' : ''}>
					{v.withinNoise ? '≈ 0' : fmtSigned(v.dpsGainPct)}
				</td>
				<td class="bar-col">
					<div class="bar"><span style="width: {v.withinNoise ? 0 : (Math.max(0, v.attackEquivalent) / max) * 100}%"></span></div>
					<span class="num eq" class:zero={v.withinNoise}>{v.withinNoise ? '–' : fmtOne(v.attackEquivalent)}</span>
				</td>
			</tr>
		{/each}
	</tbody>
</table>

<style>
	.lead {
		font-size: 0.85rem;
		margin: 0 0 0.75rem;
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
	.zero {
		color: var(--muted);
	}
	.bar-col {
		width: 38%;
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
		background: var(--accent);
		border-radius: 4px;
	}
	.eq {
		display: inline-block;
		width: 3.3rem;
		text-align: right;
	}
</style>
