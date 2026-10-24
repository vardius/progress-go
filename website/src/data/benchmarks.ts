// Results behind the benchmark charts and tables, updated together. The suite reruns with the commands on the
// Benchmarks page; the comparison comes from a harness that runs the old and new versions side by side.

export const environment = 'Apple M4 Max, Go 1.27.0, median of 5 runs';

/** The two versions the charts compare. */
export const versions = { before: 'untagged', after: 'v1' } as const;

export type Group = 'advance' | 'bar';

export interface Result {
	ns: number;
	allocs: number;
}

export interface Comparison {
	scenario: string;
	before: Result;
	after: Result;
}

/**
 * The untagged version against v1 on the same harness. Advance draws to io.Discard, so it measures the package
 * alone; a whole bar draws to /dev/null, a character device that both versions treat as a terminal, so each
 * frame costs a write as it would on a real one. The untagged Bar isn't safe from many goroutines, so there it
 * runs behind a mutex, as its callers had to.
 */
export const comparison: Record<Group, Comparison[]> = {
	advance: [
		{ scenario: '1 goroutine', before: { ns: 133.0, allocs: 1 }, after: { ns: 1.8, allocs: 0 } },
		{ scenario: '16 goroutines at once', before: { ns: 313.2, allocs: 2 }, after: { ns: 59.6, allocs: 0 } },
		{ scenario: 'Verbose, 1 goroutine', before: { ns: 23560, allocs: 13 }, after: { ns: 1.8, allocs: 0 } },
	],
	bar: [
		{ scenario: '1,000 updates', before: { ns: 936720, allocs: 2781 }, after: { ns: 4260, allocs: 6 } },
		{ scenario: '1,000 updates, verbose', before: { ns: 25020000, allocs: 10799 }, after: { ns: 4830, allocs: 8 } },
	],
};

export interface SuiteResult {
	benchmark: string;
	what: string;
	ns: number;
	bytes: number;
	allocs: number;
}

/** v1's own benchmark suite, `go test -bench .`. */
export const suite: SuiteResult[] = [
	{ benchmark: 'Advance', what: 'One atomic add', ns: 1.8, bytes: 0, allocs: 0 },
	{ benchmark: 'AdvanceParallel', what: 'One bar, advanced from 16 goroutines at once', ns: 60.7, bytes: 0, allocs: 0 },
	{ benchmark: 'Redraw', what: 'One frame on a terminal, in one Write', ns: 44.3, bytes: 0, allocs: 0 },
	{ benchmark: 'RedrawVerbose', what: 'One verbose frame: heap size and times', ns: 268.9, bytes: 0, allocs: 0 },
	{ benchmark: 'Bar', what: 'New, Start, 1,000 Advances and Stop', ns: 2000, bytes: 848, allocs: 5 },
];
