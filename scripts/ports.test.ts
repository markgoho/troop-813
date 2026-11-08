import { describe, expect, test } from "bun:test";
import { emulatorConfig, localStack, portOffset } from "./ports";

const committed = {
  firestore: { rules: "firestore.rules" },
  emulators: {
    singleProjectMode: false,
    firestore: { host: "127.0.0.1", port: 21000, websocketPort: 21001 },
    auth: { host: "127.0.0.1", port: 21002 },
    hub: { host: "127.0.0.1", port: 21003 },
    logging: { host: "127.0.0.1", port: 21004 },
    ui: { enabled: true, host: "127.0.0.1", port: 21005 },
  },
};

describe("portOffset", () => {
  test("defaults to 0 with no env and no .port-offset file", () => {
    expect(portOffset({}, null)).toBe(0);
  });

  test("reads the .port-offset file", () => {
    expect(portOffset({}, "3\n")).toBe(3);
  });

  test("PORT_OFFSET env wins over the file", () => {
    expect(portOffset({ PORT_OFFSET: "7" }, "3")).toBe(7);
  });

  test("rejects a non-integer offset", () => {
    expect(() => portOffset({ PORT_OFFSET: "abc" }, null)).toThrow(
      /PORT_OFFSET/,
    );
    expect(() => portOffset({}, "-1")).toThrow(/\.port-offset/);
  });
});

describe("localStack", () => {
  test("offset 0 uses the committed base ports", () => {
    expect(localStack({}, null).ports).toEqual({
      firestore: 21000,
      firestoreWebsocket: 21001,
      auth: 21002,
      hub: 21003,
      logging: 21004,
      ui: 21005,
      api: 21006,
      hq: 21007,
    });
  });

  test("offset 2 shifts every port by two blocks of 10", () => {
    expect(localStack({}, "2").ports).toEqual({
      firestore: 21020,
      firestoreWebsocket: 21021,
      auth: 21022,
      hub: 21023,
      logging: 21024,
      ui: 21025,
      api: 21026,
      hq: 21027,
    });
  });

  test("project ID defaults to demo-troop-813 and is overridable", () => {
    expect(localStack({}, null).projectId).toBe("demo-troop-813");
    expect(
      localStack({ DEMO_PROJECT_ID: "demo-troop-813-b" }, null).projectId,
    ).toBe("demo-troop-813-b");
  });

  test("refuses a project ID without the demo- prefix", () => {
    expect(() => localStack({ DEMO_PROJECT_ID: "troop-813" }, null)).toThrow(
      /demo-/,
    );
  });
});

describe("emulatorConfig", () => {
  test("rewrites every emulator port, including hub, logging, UI and websocket", () => {
    const config = emulatorConfig(committed, localStack({}, "1").ports);
    expect(config.emulators).toEqual({
      singleProjectMode: false,
      firestore: { host: "127.0.0.1", port: 21010, websocketPort: 21011 },
      auth: { host: "127.0.0.1", port: 21012 },
      hub: { host: "127.0.0.1", port: 21013 },
      logging: { host: "127.0.0.1", port: 21014 },
      ui: { enabled: true, host: "127.0.0.1", port: 21015 },
    });
    expect(config.firestore).toEqual({ rules: "firestore.rules" });
  });
});
