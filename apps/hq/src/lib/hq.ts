// HQ's real dependencies, shared with every Leader screen through Svelte
// context. Pages call getHq(); tests render components with fakes as props.
import { getContext, setContext } from "svelte";
import { type Api, httpApi } from "./api";
import { firebaseLive, firebaseSession } from "./firebase";
import type { Me } from "./contract.gen";
import type { Live } from "./live";
import type { Session } from "./session";

export interface Hq {
  session: Session;
  api: Api;
  live: Live;
}

const key = Symbol("hq");

/** Builds the real Session and Api and puts them in context for descendants. */
export function provideHq(): Hq {
  const session = firebaseSession();
  const hq: Hq = {
    session,
    api: httpApi(fetch, () => session.idToken()),
    live: firebaseLive(),
  };
  setContext(key, hq);
  return hq;
}

export function getHq(): Hq {
  const hq = getContext<Hq | undefined>(key);
  if (!hq) throw new Error("getHq: no provideHq() in a parent layout");
  return hq;
}

const callerKey = Symbol("caller");

/** Puts the signed-in Leader (from RoleGate) in context; LeaderShell calls it. */
export function provideCaller(me: Me): void {
  setContext(callerKey, me);
}

/** The signed-in Leader or Admin. Only inside the (leader) group. */
export function getCaller(): Me {
  const me = getContext<Me | undefined>(callerKey);
  if (!me) throw new Error("getCaller: no LeaderShell in a parent");
  return me;
}
