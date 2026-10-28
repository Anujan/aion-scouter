<script lang="ts">
	// The first seconds of one simulated fight: one lane per skill, one block
	// per cast.
	let { timeline, window = 45 }: { timeline: { t: number; skill: string; cast: number }[]; window?: number } =
		$props();

	const casts = $derived(timeline.filter((c) => c.t < window));
	const lanes = $derived([...new Set(casts.map((c) => c.skill))]);
	const ticks = $derived(Array.from({ length: Math.floor(window / 5) + 1 }, (_, i) => i * 5));
	const x = (t: number) => (t / window) * 100;
</script>

<div class="timeline">
	{#each lanes as lane (lane)}
		<div class="lane">
			<span class="label" title={lane}>{lane}</span>
			<div class="track">
				{#each casts.filter((c) => c.skill === lane) as c, i (i)}
					<span
						class="cast"
						style="left: {x(c.t)}%; width: max(3px, {x(Math.max(c.cast, 0.15))}%)"
						title="{lane} at {c.t.toFixed(1)}s ({c.cast.toFixed(2)}s cast)"
					></span>
				{/each}
			</div>
		</div>
	{/each}
	<div class="lane axis">
		<span class="label"></span>
		<div class="track">
			{#each ticks as t (t)}
				<span class="tick" style="left: {x(t)}%">{t}s</span>
			{/each}
		</div>
	</div>
</div>

<style>
	.timeline {
		font-size: 0.78rem;
	}
	.lane {
		display: grid;
		grid-template-columns: minmax(90px, 190px) 1fr;
		align-items: center;
		gap: 0.6rem;
		height: 20px;
	}
	.label {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		color: var(--muted);
		text-align: right;
	}
	.track {
		position: relative;
		height: 12px;
		background: var(--surface-2);
		border-radius: 3px;
	}
	.cast {
		position: absolute;
		top: 0;
		bottom: 0;
		background: var(--kind-skill);
		border-radius: 2px;
	}
	.axis .track {
		background: none;
	}
	.tick {
		position: absolute;
		transform: translateX(-50%);
		color: var(--muted);
		font-size: 0.7rem;
	}
	@media (max-width: 640px) {
		.lane {
			grid-template-columns: 90px 1fr;
		}
	}
</style>
