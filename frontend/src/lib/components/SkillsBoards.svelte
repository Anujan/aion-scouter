<script lang="ts">
	import type { Board, SkillInfo } from '$lib/api';

	let { skills, boards }: { skills: SkillInfo[]; boards: Board[] } = $props();

	const categories = [
		['Active', 'Active skills'],
		['Dp', 'Stigmas'],
		['Passive', 'Passives']
	] as const;
</script>

<div class="cols">
	<div class="card">
		<div class="section-title">Daevanion boards</div>
		{#each boards as b (b.id)}
			<div class="board">
				<div class="row">
					<b>{b.name}</b>
					<span class="num muted">{b.pointsSpent} / {b.pointsTotal} points</span>
				</div>
				<div class="meter"><span style="width: {b.pointsTotal ? (b.pointsSpent / b.pointsTotal) * 100 : 0}%"></span></div>
				{#if b.stats?.length}
					<div class="chips">
						{#each b.stats as s, i (i)}<span class="badge">{s.label} +{s.value}</span>{/each}
					</div>
				{/if}
			</div>
		{/each}
		<p class="muted small">The Daevanion optimizer is next on the roadmap.</p>
	</div>

	<div class="card">
		{#each categories as [cat, title] (cat)}
			{@const list = skills.filter((s) => s.category === cat && s.level > 0)}
			{#if list.length}
				<div class="section-title">{title}</div>
				<ul>
					{#each list as s (s.id)}
						<li class:off={cat !== 'Passive' && !s.equipped}>
							<span>{s.name}</span>
							<span class="num">{`Lv ${s.level}`}{#if cat !== 'Passive' && !s.equipped}<span class="muted">{' · not slotted'}</span>{/if}</span>
						</li>
					{/each}
				</ul>
			{/if}
		{/each}
	</div>
</div>

<style>
	.cols {
		display: grid;
		gap: 0.75rem;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 380px), 1fr));
		align-items: start;
	}
	.board {
		margin-bottom: 0.9rem;
	}
	.row {
		display: flex;
		justify-content: space-between;
		font-size: 0.9rem;
	}
	.meter {
		height: 6px;
		background: var(--surface-2);
		border-radius: 3px;
		margin: 0.3rem 0;
		overflow: hidden;
	}
	.meter span {
		display: block;
		height: 100%;
		background: var(--accent);
	}
	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: 0.25rem;
	}
	ul {
		list-style: none;
		padding: 0;
		margin: 0 0 1rem;
		font-size: 0.88rem;
	}
	li {
		display: flex;
		justify-content: space-between;
		padding: 0.18rem 0;
		border-bottom: 1px solid var(--surface-2);
	}
	li.off {
		color: var(--muted);
	}
	.small {
		font-size: 0.8rem;
	}
</style>
