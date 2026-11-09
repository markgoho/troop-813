// Focus hand-off for a committed action whose focused row is then removed by
// the live snapshot (a check-in, an Undo, a Deactivate, a Delete).
import { tick } from "svelte";

/** The list row (li) holding focus right now, or null. */
export function focusedRow(): Element | null {
  return document.activeElement?.closest("li") ?? null;
}

/** The first button in the row after `row`, for focus to land on. */
export function nextRowButton(row: Element | null): HTMLElement | null {
  return row?.nextElementSibling?.querySelector("button") ?? null;
}

/**
 * Once `row` has left the document, focuses the first target that exists
 * (targets are read late: a heading or field may only exist by then). Does
 * nothing when `row` is null (focus wasn't in the acted-on row, so don't
 * steal it) or never leaves.
 */
export async function focusAfterRemoval(
  row: Element | null,
  ...targets: Array<() => HTMLElement | null | undefined>
): Promise<void> {
  if (!row) return;
  // The page itself going away (navigation, unmount) is not a row leaving.
  const page = row.closest("main");

  for (let i = 0; i < 60 && row.isConnected && page?.isConnected; i++) {
    await tick();
    if (!row.isConnected) break;
    await new Promise((r) => setTimeout(r, 50));
  }
  if (row.isConnected || !page?.isConnected) return;
  await tick();
  for (const t of targets) {
    const el = t();
    if (el?.isConnected) {
      el.focus();
      return;
    }
  }
}
