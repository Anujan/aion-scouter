<script lang="ts">
	import type { Gear } from '$lib/api';

	let { gear }: { gear: Gear[] } = $props();
</script>

<div class="grid">
	{#each gear as g (g.slotPos)}
		<div class="card item grade-{g.grade?.toLowerCase()}">
			<div class="head">
				{#if g.icon}<img src={g.icon} alt="" width="36" height="36" loading="lazy" />{/if}
				<div>
					<div class="name">
						{g.name}
						{#if g.enchant}<span class="enchant">+{g.enchant}</span>{/if}
						{#if g.exceed}<span class="badge warn">Exceed {g.exceed}</span>{/if}
					</div>
					<div class="muted small">{g.slot} · {g.category} · {g.grade}</div>
				</div>
			</div>
			{#if g.main?.length}
				<ul>
					{#each g.main as s, i (i)}
						<li><span>{s.label}</span><span class="num">{s.value}{#if s.extra} <em>{s.extra}</em>{/if}</span></li>
					{/each}
				</ul>
			{/if}
			{#if g.options?.length}
				<ul class="opts">
					{#each g.options as s, i (i)}
						<li><span>{s.label}</span><span class="num">{s.value}</span></li>
					{/each}
				</ul>
			{/if}
			{#if g.manastones?.length}
				<ul class="stones">
					{#each g.manastones as s, i (i)}
						<li><span>◆ {s.label}</span><span class="num">{s.value}</span></li>
					{/each}
				</ul>
			{/if}
			{#if g.skills?.length || g.theostones?.length || g.set}
				<div class="extras small muted">
					{#each g.skills ?? [] as s, i (i)}<span>{s.label} {s.value}</span>{/each}
					{#each g.theostones ?? [] as t, i (i)}<span>{t}</span>{/each}
					{#if g.set}<span>Set: {g.set}</span>{/if}
				</div>
			{/if}
		</div>
	{/each}
</div>

<style>
	.grid {
		display: grid;
		gap: 0.6rem;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 260px), 1fr));
	}
	.item {
		padding: 0.75rem 0.85rem;
		border-left: 3px solid var(--border);
		font-size: 0.85rem;
	}
	.grade-legend {
		border-left-color: #f59f00;
	}
	.grade-unique {
		border-left-color: #e8590c;
	}
	.grade-epic {
		border-left-color: #ae3ec9;
	}
	.grade-rare {
		border-left-color: #1c7ed6;
	}
	.head {
		display: flex;
		gap: 0.6rem;
		align-items: center;
		margin-bottom: 0.4rem;
	}
	.head img {
		border-radius: 6px;
	}
	.name {
		font-weight: 650;
	}
	.enchant {
		color: var(--accent);
		margin-left: 0.25rem;
	}
	.small {
		font-size: 0.78rem;
	}
	ul {
		list-style: none;
		margin: 0.3rem 0 0;
		padding: 0;
	}
	li {
		display: flex;
		justify-content: space-between;
		gap: 0.5rem;
	}
	em {
		font-style: normal;
		color: var(--good);
	}
	.opts {
		border-top: 1px dashed var(--border);
		padding-top: 0.3rem;
	}
	.stones {
		color: var(--muted);
	}
	.extras {
		display: flex;
		flex-wrap: wrap;
		gap: 0.2rem 0.6rem;
		margin-top: 0.4rem;
	}
</style>
