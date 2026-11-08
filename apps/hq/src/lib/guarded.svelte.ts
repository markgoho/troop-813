// One guarded action at a time for a Leader screen (Tonight, a Meeting's
// page): `run` ignores a second tap while one call is in flight, clears the
// last message, and turns a failure into `message`. A 404 means someone
// else changed the thing first, so the message says so.
import { ApiError } from "./api";

export interface Guard {
  /** True while a call is in flight: disable the buttons. */
  readonly busy: boolean;
  /** The last failure, or null. */
  readonly message: string | null;
  /**
   * Runs `action`; true when it succeeded, false when it failed or another
   * action was in flight. On failure `message` becomes `failure` (plus "It
   * may already have changed." for a 404), or `explain(err)` when given.
   */
  run(
    failure: string,
    action: () => Promise<unknown>,
    explain?: (err: unknown) => string,
  ): Promise<boolean>;
}

export function guardedActions(): Guard {
  let busy = $state(false);
  let message = $state<string | null>(null);
  return {
    get busy() {
      return busy;
    },
    get message() {
      return message;
    },
    async run(failure, action, explain) {
      if (busy) return false;
      busy = true;
      message = null;
      try {
        await action();
        return true;
      } catch (err) {
        message = explain
          ? explain(err)
          : err instanceof ApiError && err.status === 404
            ? `${failure} It may already have changed.`
            : failure;
        return false;
      } finally {
        busy = false;
      }
    },
  };
}
