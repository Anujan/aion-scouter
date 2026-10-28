// Display formatting only. No game math lives in the frontend.

const int = new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 });
const one = new Intl.NumberFormat('en-US', { maximumFractionDigits: 1 });
const compact = new Intl.NumberFormat('en-US', { notation: 'compact', maximumFractionDigits: 1 });

export const fmtInt = (v: number) => int.format(v);
export const fmtOne = (v: number) => one.format(v);
export const fmtCompact = (v: number) => compact.format(v);

/** A fraction (0.25) as a percentage ("25.0%"). */
export const fmtFrac = (v: number, digits = 1) => `${(v * 100).toFixed(digits)}%`;

/** A stat value, with % for percentage stats. */
export const fmtStat = (v: number, pct: boolean) => (pct ? `${one.format(v)}%` : int.format(v));

export const fmtSigned = (v: number, digits = 2) => `${v >= 0 ? '+' : ''}${v.toFixed(digits)}%`;

export const kindLabel: Record<string, string> = {
	skill: 'Skill',
	dot: 'Damage over time',
	spirit: 'Spirit',
	proc: 'Proc'
};
