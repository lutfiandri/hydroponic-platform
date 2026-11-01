const INTERVAL_MS = 10_000;
const DEVICE_CODE = "tank-01";

const { execFile } = require("node:child_process");
const path = require("node:path");
const { promisify } = require("node:util");

const exec = promisify(execFile);
let sending = false;

async function publish(kind, payload) {
  const topic = `devices/${DEVICE_CODE}/${kind}`;
  try {
    await exec("docker", [
      "compose", "exec", "-T", "mqtt", "mosquitto_pub",
      "-h", "localhost", "-q", "1", "-t", topic, "-m", JSON.stringify(payload),
    ], { cwd: path.resolve(__dirname, ".."), timeout: 5_000 });
    console.log(`[${new Date().toISOString()}] ${topic}`, payload);
  } catch (error) {
    console.error(`Publish gagal: ${topic}: ${error.stderr?.trim() || error.message}`);
  }
}

async function sendMockData() {
  if (sending) return;
  sending = true;
  try {
    const timestamp = new Date().toISOString();
    await Promise.all([
      publish("reading", {
        recorded_at: timestamp,
        water_level: Math.round(Math.random() * 100),
        ppm: 600 + Math.floor(Math.random() * 601),
        water_temperature: Math.round((24 + Math.random() * 6) * 10) / 10,
        battery_percentage: 50 + Math.floor(Math.random() * 51),
      }),
      publish("event", {
        occurred_at: timestamp,
        event_type: "mock_heartbeat",
        source: "system",
        status: "success",
        message: "Mock device heartbeat",
      }),
    ]);
  } finally {
    sending = false;
  }
}

const timer = setInterval(sendMockData, INTERVAL_MS);
for (const signal of ["SIGINT", "SIGTERM"]) {
  process.once(signal, () => clearInterval(timer));
}
sendMockData();
