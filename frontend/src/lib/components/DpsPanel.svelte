<script lang="ts">
	import { simulate, type Character, type Meta, type Scenario, type SimResponse } from '$lib/api';
	import { fmtFrac, fmtInt, fmtOne } from '$lib/format';
	import DamageBreakdown from './DamageBreakdown.svelte';
	import Efficiency from './Efficiency.svelte';
	import Timeline from './Timeline.svelte';
	import { untrack } from 'svelte';

	let { character, meta }: { character: Character; meta: Meta } = $props();

	// The panel is keyed by character, so taking meta's default once is intended.
	let scenario = $state<Scenario>(untrack(() => ({ ...meta.defaultScenario })));
	let overrides = $state<Record<string, string>>({});
	let response = $state<SimResponse>();
	let loading = $state(false);
	let error = $state('');

	const target = $derived(meta.targets.find((t) => t.id === scenario.targetId));
	const rebuilt = $derived(Object.fromEntries(character.stats.map((s) => [s.key, s.value])));
	const activeOverrides = $derived(
		Object.fromEntries(
			Object.entries(overrides)
				.filter(([, v]) => v.trim() !== '' && !Number.isNaN(Number(v)))
				.map(([k, v]) => [k, Number(v)])
		)
	);

	async function run() {
		loading = true;
		error = '';
		try {
			response = await simulate({
				region: character.ref.region,
				serverId: character.ref.serverId,
				characterId: character.ref.characterId,
				scenario,
				overrides: activeOverrides,
				efficiency: true
			});
		} catch (e) {
			error = (e as Error).message;
		} finally {
			loading = false;
		}
	}

	// Re-run whenever the scenario changes (overrides apply on the button).
	$effect(() => {
		void scenario.targetId;
		void scenario.duration;
		void scenario.powerShard;
		untrack(run);
	});

	const r = $derived(response?.result);
</script>

<div class="controls card">
	<label>
		<span>Target</span>
		<select bind:value={scenario.targetId}>
			{#each meta.targets as t (t.id)}<option value={t.id}>{t.name}</option>{/each}
		</select>
	</label>
	<label>
		<span>Fight length</span>
		<select bind:value={scenario.duration}>
			<option value={60}>1 min</option>
			<option value={120}>2 min</option>
			<option value={180}>3 min</option>
			<option value={300}>5 min</option>
		</select>
	</label>
	<label class="check">
		<input type="checkbox" bind:checked={scenario.powerShard} />
		<span>Power Shard active</span>
	</label>
	{#if loading}<span class="muted">Simulating…</span>{/if}
	{#if error}<span class="error">{error}</span>{/if}
</div>

{#if target?.estimated}
	<p class="note"><b>{target.name}:</b> {target.description}</p>
{/if}

{#if r}
	<div class="headline">
		<div class="card big">
			<div class="section-title">Estimated DPS</div>
			<div class="dps num">{fmtInt(r.dps)}</div>
			<div class="muted">
				{fmtInt(r.totalDamage)} damage in {r.duration}s against {r.target.name}, averaged over {r.iterations} runs
			</div>
		</div>
		<div class="card summary">
			<div class="section-title">Why this number</div>
			<dl>
				<dt>Effective Attack</dt><dd class="num">{fmtInt(r.summary.effectiveAttack)}</dd>
				<dt>Crit chance</dt><dd class="num">{fmtFrac(r.summary.critChance)}</dd>
				<dt>Crit multiplier</dt><dd class="num">×{r.summary.critMultiplier.toFixed(2)}</dd>
				<dt>Damage Boost (total)</dt><dd class="num">+{fmtFrac(r.summary.damageBoost)}</dd>
				<dt>Double chance (after resist)</dt><dd class="num">{fmtFrac(r.summary.doubleChance)}</dd>
				<dt>Combat Speed</dt><dd class="num">+{fmtFrac(r.summary.combatSpeed)}</dd>
				<dt>Cooldown Reduction</dt><dd class="num">{fmtFrac(r.summary.cooldownReduction)}</dd>
				<dt>Lost to target defense</dt><dd class="num">{fmtOne(r.summary.defenseReduction)} per hit</dd>
			</dl>
		</div>
	</div>

	<div class="card">
		<div class="section-title">Damage breakdown</div>
		<DamageBreakdown sources={r.sources} />
	</div>

	<div class="two">
		<div class="card">
			<div class="section-title">What to raise next</div>
			{#if response?.efficiency}<Efficiency values={response.efficiency} />{/if}
		</div>
		<div class="card">
			<div class="section-title">Rotation, first 45 seconds</div>
			<Timeline timeline={r.timeline} />
		</div>
	</div>

	<details class="card">
		<summary><b>Match your stat window</b> <span class="muted">Correct any stat that differs from your in-game window</span></summary>
		<p class="muted small">
			The rebuild can miss sources the API doesn't expose, such as class base stats, wings and
			buffs. Type the value from your in-game stat window to override it, then re-run.
		</p>
		<div class="overrides">
			{#each meta.overrideStats as o (o.key)}
				<label>
					<span>{o.label}{o.pct ? ' (%)' : ''}</span>
					<input
						inputmode="decimal"
						placeholder={fmtOne(rebuilt[o.key] ?? 0)}
						bind:value={overrides[o.key]}
					/>
				</label>
			{/each}
		</div>
		<button class="primary" onclick={run} disabled={loading}>Re-run with my stats</button>
	</details>

	<details class="card">
		<summary><b>Assumptions</b> <span class="muted">What this estimate takes for granted</span></summary>
		<ul class="small">
			{#each r.assumptions as a, i (i)}<li>{a}</li>{/each}
			<li>
				Damage per hit uses the community-tested formula: Attack, then Weapon Damage Boost, Attack
				Bonus and Attack Increase, then skill coefficient, minus 10% of (Defense − Penetration),
				then the Damage Boost pool, crit, Double and Multi-hit as expected values.
			</li>
			{#if target}<li>Target: {target.source}</li>{/if}
		</ul>
		{#if character.report.unparsed?.length}
			<p class="small muted">Not counted (effects we can't model yet): {character.report.unparsed.join('; ')}</p>
		{/if}
		{#if character.report.conditional?.length}
			<p class="small muted">Counted as always on: {character.report.conditional.join('; ')}</p>
		{/if}
	</details>
{/if}

<style>
	.controls {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		gap: 0.75rem 1.25rem;
		margin-bottom: 0.75rem;
	}
	label {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
		font-size: 0.8rem;
		color: var(--muted);
	}
	label.check {
		flex-direction: row;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.9rem;
		color: var(--text);
		padding-bottom: 0.5rem;
	}
	.note {
		font-size: 0.85rem;
		background: var(--warn-soft);
		color: var(--text);
		border-radius: var(--radius);
		padding: 0.6rem 0.9rem;
		margin: 0 0 0.75rem;
	}
	.headline {
		display: grid;
		gap: 0.75rem;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 320px), 1fr));
		margin-bottom: 0.75rem;
	}
	.dps {
		font-size: clamp(2.4rem, 7vw, 3.6rem);
		font-weight: 800;
		letter-spacing: -0.03em;
		line-height: 1.05;
	}
	dl {
		display: grid;
		grid-template-columns: 1fr auto;
		gap: 0.2rem 1rem;
		margin: 0;
		font-size: 0.88rem;
	}
	dt {
		color: var(--muted);
	}
	dd {
		margin: 0;
		font-weight: 600;
		text-align: right;
	}
	.two {
		display: grid;
		gap: 0.75rem;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 460px), 1fr));
		margin: 0.75rem 0;
		align-items: start;
	}
	details {
		margin-top: 0.75rem;
	}
	summary {
		cursor: pointer;
	}
	.small {
		font-size: 0.85rem;
	}
	.overrides {
		display: grid;
		gap: 0.5rem 0.9rem;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 180px), 1fr));
		margin: 0.75rem 0;
	}
</style>
