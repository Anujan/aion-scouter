// Typed client for the Go API. These types mirror the backend's JSON; the
// frontend only displays them.

export interface Region {
	id: string;
	label: string;
}

export interface Target {
	id: string;
	name: string;
	description: string;
	boss: boolean;
	defense: number;
	critResist: number;
	critDamageTolerance: number;
	damageTolerance: number;
	doubleResist: number;
	estimated: boolean;
	source: string;
}

export interface ClassInfo {
	slug: string;
	supported: boolean;
	status: string;
	assumptions: string[];
}

export interface Scenario {
	targetId: string;
	duration: number;
	iterations: number;
	powerShard: boolean;
	seed: number;
}

export interface OverrideStat {
	key: string;
	label: string;
	pct: boolean;
}

export interface Meta {
	regions: Region[];
	targets: Target[];
	classes: ClassInfo[];
	constants: { notes: Record<string, string> } & Record<string, unknown>;
	defaultScenario: Scenario;
	overrideStats: OverrideStat[];
}

export interface SearchRow {
	characterId: string;
	name: string;
	level: number;
	serverId: number;
	serverName: string;
	region: string;
	className: string;
	supported: boolean;
	profileImageUrl: string;
}

export interface Contribution {
	source: string;
	detail: string;
	value: number;
}

export interface StatEntry {
	key: string;
	label: string;
	group: string;
	pct: boolean;
	value: number;
	sources: Contribution[];
}

export interface StatLine {
	label: string;
	value: string;
	extra?: string;
}

export interface Gear {
	slot: string;
	slotPos: number;
	name: string;
	grade: string;
	category: string;
	enchant: number;
	maxEnchant: number;
	exceed: number;
	icon?: string;
	main: StatLine[] | null;
	options: StatLine[] | null;
	manastones: StatLine[] | null;
	skills: StatLine[] | null;
	theostones: string[] | null;
	set?: string;
}

export interface SkillInfo {
	id: number;
	name: string;
	category: string;
	level: number;
	equipped: boolean;
	icon?: string;
}

export interface Board {
	id: number;
	name: string;
	openNodes: number;
	totalNodes: number;
	pointsSpent: number;
	pointsTotal: number;
	stats: StatLine[] | null;
}

export interface Character {
	ref: { region: string; serverId: number; characterId: string };
	fetchedAt: string;
	profile: {
		name: string;
		level: number;
		className: string;
		server: string;
		race: string;
		title: string;
		combatPower: number;
		itemLevel: number;
		profileImage: string;
	};
	class: ClassInfo;
	equipment: Gear[];
	skills: SkillInfo[];
	boards: Board[];
	stats: StatEntry[];
	report: { unparsed: string[] | null; conditional: string[] | null; procs: string[] | null };
}

export interface Source {
	name: string;
	kind: 'skill' | 'dot' | 'spirit' | 'proc';
	damage: number;
	share: number;
	dps: number;
	casts: number;
	hits: number;
	avgHit: number;
}

export interface SimResult {
	dps: number;
	dpsStdErr: number;
	totalDamage: number;
	duration: number;
	iterations: number;
	target: Target;
	summary: {
		effectiveAttack: number;
		critChance: number;
		critMultiplier: number;
		damageBoost: number;
		doubleChance: number;
		multiHitChance: number;
		combatSpeed: number;
		cooldownReduction: number;
		defenseReduction: number;
	};
	sources: Source[];
	timeline: { t: number; skill: string; cast: number }[];
	assumptions: string[];
}

export interface StatValue {
	key: string;
	label: string;
	step: number;
	pct: boolean;
	dpsGain: number;
	dpsGainPct: number;
	attackEquivalent: number;
	withinNoise: boolean;
}

export interface SimResponse {
	result: SimResult;
	efficiency?: StatValue[];
	scenario: Scenario;
}

export class ApiError extends Error {
	constructor(
		public status: number,
		message: string
	) {
		super(message);
	}
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(path, init);
	const body = await res.json().catch(() => ({}));
	if (!res.ok) throw new ApiError(res.status, body.error ?? `Request failed (${res.status})`);
	return body as T;
}

let metaPromise: Promise<Meta> | undefined;
export const getMeta = () => (metaPromise ??= request<Meta>('/api/meta'));

export const search = (region: string, name: string) =>
	request<{ results: SearchRow[] }>(
		`/api/search?${new URLSearchParams({ region, name })}`
	).then((r) => r.results);

export const getCharacter = (region: string, serverId: string | number, characterId: string) =>
	request<Character>(
		`/api/character/${region}/${serverId}/${encodeURIComponent(characterId)}`
	);

export const simulate = (body: {
	region: string;
	serverId: number;
	characterId: string;
	scenario: Scenario;
	overrides?: Record<string, number>;
	efficiency?: boolean;
}) =>
	request<SimResponse>('/api/simulate', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(body)
	});

export const characterPath = (region: string, serverId: number, characterId: string) =>
	`/c/${region}/${serverId}/${encodeURIComponent(characterId)}`;
