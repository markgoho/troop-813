// HQ's real dependencies, shared with every Leader screen through Svelte
// context. Pages call getHq(); tests render components with fakes as props.
import { getContext, setContext } from "svelte";
import { type Api, httpApi } from "./api";
import { firebaseSession } from "./firebase";
import type { Session } from "./session";

export interface Hq {
  session: Session;
  api: Api;
}

const key = Symbol("hq");

/** Builds the real Session and Api and puts them in context for descendants. */
export function provideHq(): Hq {
  const session = firebaseSession();
  const hq: Hq = { session, api: httpApi(fetch, () => session.idToken()) };
  setContext(key, hq);
  return hq;
}

export function getHq(): Hq {
  const hq = getContext<Hq | undefined>(key);
  if (!hq) throw new Error("getHq: no provideHq() in a parent layout");
  return hq;
}
