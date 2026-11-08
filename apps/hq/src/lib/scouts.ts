// Scout name helpers shared by the Leader screens.
import type { Scout } from "./contract.gen";

/** "Avery Quill": the name as Leaders see and label it. */
export const fullName = (s: Pick<Scout, "firstName" | "lastName">) =>
  `${s.firstName} ${s.lastName}`;

/** Sort order for Scout lists: first name, then last. */
export const byName = (
  a: Pick<Scout, "firstName" | "lastName">,
  b: Pick<Scout, "firstName" | "lastName">,
) =>
  a.firstName.localeCompare(b.firstName) ||
  a.lastName.localeCompare(b.lastName);
