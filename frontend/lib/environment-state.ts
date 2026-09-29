/**
 * Single source of truth for environment / sandbox status rendering.
 *
 * Two different state machines are still live in the API:
 *
 *   - environments (`/api/environments`, `/api/projects/:id` child routes) use
 *     the 12-value set from backend/internal/domain/environment.go
 *   - the legacy sandboxes table (`/api/projects/:id/sandboxes`) still uses the
 *     6-value set from backend/internal/domain/sandbox.go
 *
 * Previously each page carried its own partial map keyed on the old 6 values,
 * so a real environment in CREATED, BUILD_FAILED, CRASHED, SUSPENDED and the
 * rest all fell through to a grey "Idle" badge, the build-logs tab never opened
 * on a real failure, and the Restart button stayed hidden on a crashed
 * environment. Everything now resolves through here.
 */

export type EnvState = string;

export type StateTone = "running" | "building" | "stopped" | "failed" | "idle";

export interface StatePresentation {
  label: string;
  /** Tailwind classes for a status pill. */
  color: string;
  /** Tailwind class for the status dot. */
  dot: string;
  tone: StateTone;
}

const TONES: Record<StateTone, Omit<StatePresentation, "label">> = {
  running: {
    color: "text-emerald-400 bg-emerald-400/10 border-emerald-400/20",
    dot: "bg-emerald-400 animate-pulse shadow-[0_0_6px_#34d399]",
    tone: "running",
  },
  building: {
    color: "text-blue-400 bg-blue-400/10 border-blue-400/20",
    dot: "bg-blue-400 animate-pulse",
    tone: "building",
  },
  stopped: {
    color: "text-orange-400 bg-orange-400/10 border-orange-400/20",
    dot: "bg-orange-400",
    tone: "stopped",
  },
  failed: {
    color: "text-red-400 bg-red-400/10 border-red-400/20",
    dot: "bg-red-400",
    tone: "failed",
  },
  idle: {
    color: "text-gray-400 bg-gray-400/10 border-gray-400/20",
    dot: "bg-gray-400",
    tone: "idle",
  },
};

const PRESENTATION: Record<EnvState, StatePresentation> = {
  // --- environment states (domain.EnvironmentState) ---
  CREATED: { label: "Queued", ...TONES.building },
  BUILDING: { label: "Building", ...TONES.building },
  BUILD_FAILED: { label: "Build failed", ...TONES.failed },
  READY: { label: "Ready", ...TONES.idle },
  STARTING: { label: "Starting", ...TONES.building },
  RUNNING: { label: "Running", ...TONES.running },
  STOPPING: { label: "Stopping", ...TONES.stopped },
  STOPPED: { label: "Stopped", ...TONES.stopped },
  SUSPENDING: { label: "Suspending", ...TONES.stopped },
  SUSPENDED: { label: "Suspended", ...TONES.stopped },
  CRASHED: { label: "Crashed", ...TONES.failed },
  DELETING: { label: "Deleting", ...TONES.stopped },

  // --- legacy sandbox states (domain.SandboxState) ---
  IDLE: { label: "Idle", ...TONES.idle },
  PENDING: { label: "Queued", ...TONES.building },
  FAILED: { label: "Failed", ...TONES.failed },
};

const UNKNOWN: StatePresentation = { label: "Unknown", ...TONES.idle };

export function presentState(state: EnvState | undefined | null): StatePresentation {
  if (!state) return UNKNOWN;
  return PRESENTATION[state] ?? { ...UNKNOWN, label: state };
}

/** True while the environment is being built and a log tail is worth showing. */
export function isBuilding(state: EnvState | undefined | null): boolean {
  return state === "BUILDING" || state === "PENDING" || state === "CREATED" || state === "STARTING";
}

/** True when a build has failed, so the build-logs tab should surface itself. */
export function isFailed(state: EnvState | undefined | null): boolean {
  return state === "BUILD_FAILED" || state === "FAILED" || state === "CRASHED";
}

/** True when a container exists, so preview/editor affordances make sense. */
export function hasContainer(state: EnvState | undefined | null): boolean {
  return state === "RUNNING" || state === "STOPPED" || state === "SUSPENDED";
}

/** True when the environment can be started again. */
export function canStart(state: EnvState | undefined | null): boolean {
  return (
    state === "STOPPED" || state === "SUSPENDED" || state === "CRASHED" || state === "READY" || state === "FAILED"
  );
}
