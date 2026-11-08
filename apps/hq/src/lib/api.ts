// Typed client for the Go API. Components take an Api so tests can pass a fake.
// Body types come from contract.gen.ts (generated from Go, ADR 0002): never hand-write them.
import type {
  AddScoutRequest,
  AddScoutResponse,
  CheckInResponse,
  ErrorBody,
  EventCreated,
  EventInput,
  Health,
  KioskCheckInRequest,
  LeaderCheckInRequest,
  LeaderList,
  Me,
  RSVPAnswer,
  SetRole,
  SetRSVP,
  StartMeetingResponse,
  UserList,
} from "./contract.gen";

export interface Api {
  health(): Promise<Health>;
  /** Who the caller is. The first call after a first sign-in creates their users doc. */
  me(): Promise<Me>;
  /** Admin only: everyone who has signed in, newest first. */
  users(): Promise<UserList>;
  /** Admin only: Parent↔Leader. Applies on that person's next request. */
  setRole(uid: string, role: "parent" | "leader"): Promise<void>;
  /** Adds an Event (not cancelled); resolves to its id. */
  createEvent(input: EventInput): Promise<EventCreated>;
  /** Replaces an Event's name, dates and location; keeps cancelled. */
  updateEvent(id: string, input: EventInput): Promise<void>;
  setEventCancelled(id: string, cancelled: boolean): Promise<void>;
  /** Refused with 409 in_use while the Event has any RSVP doc. */
  deleteEvent(id: string): Promise<void>;
  /** A Leader sets, changes or (null) clears a Scout's RSVP. */
  setRsvp(
    eventId: string,
    scoutId: string,
    answer: RSVPAnswer | null,
  ): Promise<void>;
  /** Leader only: adds an active Scout. ApiError 409 duplicate_scout for an active exact match. */
  addScout(scout: AddScoutRequest): Promise<AddScoutResponse>;
  /**
   * Admin only: starts (or re-opens) today's Meeting and issues a new Kiosk
   * session. ApiError 409 too_late from 22:00 New York time.
   */
  startMeeting(): Promise<StartMeetingResponse>;
  /**
   * Kiosk only: checks a Scout in to the session's Meeting. Repeat check-ins
   * return the existing attendanceId. ApiError 401 session_asleep / session_expired.
   */
  kioskCheckIn(req: KioskCheckInRequest): Promise<CheckInResponse>;
  /** Leader only: checks a Scout in to a Meeting (method leader). A repeat returns the existing id. */
  leaderCheckIn(meetingId: string, scoutId: string): Promise<CheckInResponse>;
  /** Leader only: soft-deletes a check-in; the Scout can check in again. Idempotent. */
  undoCheckIn(meetingId: string, attendanceId: string): Promise<void>;
  /** Leader only: every Leader and the Admin by uid and name (no emails). */
  leaders(): Promise<LeaderList>;
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly body: ErrorBody | null,
  ) {
    super(`API ${status}${body ? `: ${body.code}` : ""}`);
  }
}

/** Returns the current ID token, or null when signed out (then requests go without one). */
export type TokenSource = () => Promise<string | null>;

async function send(
  fetchFn: typeof fetch,
  token: TokenSource,
  method: string,
  path: string,
  body?: unknown,
): Promise<Response> {
  const headers: Record<string, string> = { Accept: "application/json" };
  const t = await token();
  if (t) headers.Authorization = `Bearer ${t}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const res = await fetchFn(path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!res.ok) {
    const err = (await res.json().catch(() => null)) as ErrorBody | null;
    throw new ApiError(res.status, err);
  }
  return res;
}

async function getJSON<T>(
  fetchFn: typeof fetch,
  token: TokenSource,
  path: string,
): Promise<T> {
  const res = await send(fetchFn, token, "GET", path);
  return (await res.json()) as T;
}

/** The real client: same-origin `/api` (Vite proxies it in dev, Hosting rewrites it in prod). */
export function httpApi(
  fetchFn: typeof fetch = fetch,
  token: TokenSource = async () => null,
): Api {
  return {
    health: () => getJSON<Health>(fetchFn, token, "/api/health"),
    me: () => getJSON<Me>(fetchFn, token, "/api/me"),
    users: () => getJSON<UserList>(fetchFn, token, "/api/users"),
    setRole: async (uid, role) => {
      const body: SetRole = { role };
      await send(
        fetchFn,
        token,
        "PUT",
        `/api/users/${encodeURIComponent(uid)}/role`,
        body,
      );
    },
    createEvent: async (input) => {
      const res = await send(fetchFn, token, "POST", "/api/events", input);
      return (await res.json()) as EventCreated;
    },
    updateEvent: async (id, input) => {
      await send(fetchFn, token, "PUT", eventPath(id), input);
    },
    setEventCancelled: async (id, cancelled) => {
      const action = cancelled ? "cancel" : "uncancel";
      await send(fetchFn, token, "POST", `${eventPath(id)}/${action}`);
    },
    deleteEvent: async (id) => {
      await send(fetchFn, token, "DELETE", eventPath(id));
    },
    setRsvp: async (eventId, scoutId, answer) => {
      const body: SetRSVP = { answer };
      const path = `${eventPath(eventId)}/rsvps/${encodeURIComponent(scoutId)}`;
      await send(fetchFn, token, "PUT", path, body);
    },
    addScout: async (scout) => {
      const res = await send(fetchFn, token, "POST", "/api/scouts", scout);
      return (await res.json()) as AddScoutResponse;
    },
    startMeeting: async () => {
      const res = await send(fetchFn, token, "POST", "/api/kiosk/start");
      return (await res.json()) as StartMeetingResponse;
    },
    kioskCheckIn: async (req) => {
      const res = await send(
        fetchFn,
        token,
        "POST",
        "/api/kiosk/check-in",
        req,
      );
      return (await res.json()) as CheckInResponse;
    },
    leaderCheckIn: async (meetingId, scoutId) => {
      const body: LeaderCheckInRequest = { scoutId };
      const res = await send(
        fetchFn,
        token,
        "POST",
        `${meetingPath(meetingId)}/check-ins`,
        body,
      );
      return (await res.json()) as CheckInResponse;
    },
    undoCheckIn: async (meetingId, attendanceId) => {
      await send(
        fetchFn,
        token,
        "POST",
        `${meetingPath(meetingId)}/attendance/${encodeURIComponent(attendanceId)}/undo`,
      );
    },
    leaders: () => getJSON<LeaderList>(fetchFn, token, "/api/leaders"),
  };
}

function meetingPath(id: string): string {
  return `/api/meetings/${encodeURIComponent(id)}`;
}

function eventPath(id: string): string {
  return `/api/events/${encodeURIComponent(id)}`;
}
