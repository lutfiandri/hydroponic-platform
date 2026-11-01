const assert = require("node:assert/strict");
const { spawn } = require("node:child_process");
const { once } = require("node:events");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { setTimeout: sleep } = require("node:timers/promises");
const { test } = require("node:test");

const root = path.resolve(__dirname, "..");

function startPublisher(t, fail = false) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "mock-mqtt-"));
  const output = path.join(directory, "messages.jsonl");
  fs.writeFileSync(path.join(directory, "docker"), `#!${process.execPath}
const fs = require("node:fs");
fs.appendFileSync(${JSON.stringify(output)}, JSON.stringify({ args: process.argv.slice(2), cwd: process.cwd(), time: Date.now() }) + "\\n");
if (${fail}) { console.error("broker unavailable"); process.exit(1); }
`, { mode: 0o755 });
  const child = spawn(process.execPath, [path.join(__dirname, "mock-mqtt.js")], {
    cwd: os.tmpdir(),
    env: { ...process.env, PATH: `${directory}${path.delimiter}${process.env.PATH}` },
    stdio: ["ignore", "pipe", "pipe"],
  });
  const exit = once(child, "exit");
  let errors = "";
  child.stdout.resume();
  child.stderr.on("data", (data) => { errors += data; });
  t.after(async () => {
    if (child.exitCode === null && child.signalCode === null) child.kill("SIGTERM");
    await exit;
    fs.rmSync(directory, { recursive: true, force: true });
  });
  return {
    child,
    exit,
    errors: () => errors,
    async waitForMessages(count) {
      const deadline = Date.now() + 15_000;
      while (Date.now() < deadline) {
        assert.equal(child.exitCode, null, `publisher exited: ${errors}`);
        if (fs.existsSync(output)) {
          const messages = fs.readFileSync(output, "utf8").trim().split("\n").filter(Boolean).map(JSON.parse);
          if (messages.length >= count) return messages;
        }
        await sleep(25);
      }
      assert.fail(`expected ${count} publishes: ${errors}`);
    },
  };
}

test("publishes valid reading/event immediately and again after ten seconds", { timeout: 20_000 }, async (t) => {
  const publisher = startPublisher(t);
  const messages = await publisher.waitForMessages(4);
  assert.equal(messages.length, 4);
  for (const message of messages) {
    assert.equal(message.cwd, root);
    assert.deepEqual(message.args.slice(0, 12), ["compose", "exec", "-T", "mqtt", "mosquitto_pub", "-h", "localhost", "-q", "1", "-t", message.args[10], "-m"]);
    assert.equal(message.args.length, 13);
    const payload = JSON.parse(message.args[12]);
    assert.equal(Object.hasOwn(payload, "device_id"), false);
    if (message.args[10] === "devices/tank-01/reading") {
      assert.ok(Number.isFinite(Date.parse(payload.recorded_at)));
      for (const field of ["water_level", "battery_percentage"]) {
        assert.ok(payload[field] >= 0 && payload[field] <= 100);
      }
      assert.ok(payload.ppm >= 0);
      assert.ok(Number.isFinite(payload.water_temperature));
    } else {
      assert.equal(message.args[10], "devices/tank-01/event");
      assert.ok(Number.isFinite(Date.parse(payload.occurred_at)));
      assert.equal(payload.event_type, "mock_heartbeat");
      assert.equal(payload.source, "system");
      assert.equal(payload.status, "success");
    }
  }
  for (const topic of ["devices/tank-01/reading", "devices/tank-01/event"]) {
    const sends = messages.filter((message) => message.args[10] === topic);
    assert.equal(sends.length, 2);
    assert.ok(sends[1].time - sends[0].time >= 9_500);
    assert.notEqual(JSON.parse(sends[0].args[12]).recorded_at ?? JSON.parse(sends[0].args[12]).occurred_at,
      JSON.parse(sends[1].args[12]).recorded_at ?? JSON.parse(sends[1].args[12]).occurred_at);
  }
  publisher.child.kill("SIGINT");
  assert.deepEqual(await publisher.exit, [0, null]);
  assert.equal(publisher.errors(), "");
});

test("logs broker failures without crashing and stops cleanly on SIGTERM", { timeout: 5_000 }, async (t) => {
  const publisher = startPublisher(t, true);
  await publisher.waitForMessages(2);
  for (let attempt = 0; attempt < 100 && !publisher.errors().includes("broker unavailable"); attempt++) await sleep(10);
  assert.match(publisher.errors(), /broker unavailable/);
  assert.equal(publisher.child.exitCode, null);
  publisher.child.kill("SIGTERM");
  assert.deepEqual(await publisher.exit, [0, null]);
});
