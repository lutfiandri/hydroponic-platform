migrate((app) => {
  const zone = new Collection({
    name: "zone",
    type: "base",
    listRule: null,
    viewRule: null,
    createRule: null,
    updateRule: null,
    deleteRule: null,
    fields: [
      { name: "name", type: "text", required: true },
      { name: "description", type: "text" },
      { name: "created", type: "autodate", onCreate: true },
      { name: "updated", type: "autodate", onCreate: true, onUpdate: true },
    ],
  });
  app.save(zone);

  const device = new Collection({
    name: "device",
    type: "base",
    listRule: null,
    viewRule: null,
    createRule: null,
    updateRule: null,
    deleteRule: null,
    fields: [
      { name: "zone_id", type: "relation", collectionId: zone.id, required: true, maxSelect: 1, cascadeDelete: false },
      { name: "name", type: "text", required: true },
      { name: "device_code", type: "text", required: true },
      { name: "type", type: "select", required: true, maxSelect: 1, values: ["tank", "table"] },
      { name: "status", type: "select", maxSelect: 1, values: ["online", "offline"] },
      { name: "last_seen_at", type: "date" },
      { name: "created", type: "autodate", onCreate: true },
      { name: "updated", type: "autodate", onCreate: true, onUpdate: true },
    ],
    indexes: [
      "CREATE UNIQUE INDEX idx_device_device_code ON device (device_code)",
    ],
  });
  app.save(device);

  const reading = new Collection({
    name: "device_reading",
    type: "base",
    listRule: null,
    viewRule: null,
    createRule: null,
    updateRule: null,
    deleteRule: null,
    fields: [
      { name: "device_id", type: "relation", collectionId: device.id, required: true, maxSelect: 1, cascadeDelete: false },
      { name: "water_level", type: "number", min: 0, max: 100 },
      { name: "has_water_level", type: "bool" },
      { name: "ppm", type: "number", min: 0 },
      { name: "has_ppm", type: "bool" },
      { name: "water_temperature", type: "number" },
      { name: "has_water_temperature", type: "bool" },
      { name: "battery_percentage", type: "number", min: 0, max: 100 },
      { name: "has_battery_percentage", type: "bool" },
      { name: "recorded_at", type: "date", required: true },
      { name: "created", type: "autodate", onCreate: true },
      { name: "updated", type: "autodate", onCreate: true, onUpdate: true },
    ],
    indexes: [
      "CREATE INDEX idx_device_reading_device_id_recorded_at ON device_reading (device_id, recorded_at)",
    ],
  });
  app.save(reading);

  const event = new Collection({
    name: "device_event",
    type: "base",
    listRule: null,
    viewRule: null,
    createRule: null,
    updateRule: null,
    deleteRule: null,
    fields: [
      { name: "device_id", type: "relation", collectionId: device.id, required: true, maxSelect: 1, cascadeDelete: false },
      { name: "event_type", type: "text", required: true },
      { name: "source", type: "select", maxSelect: 1, values: ["automatic", "manual", "system"] },
      { name: "status", type: "select", maxSelect: 1, values: ["pending", "success", "failed"] },
      { name: "message", type: "text" },
      { name: "occurred_at", type: "date", required: true },
      { name: "created", type: "autodate", onCreate: true },
      { name: "updated", type: "autodate", onCreate: true, onUpdate: true },
    ],
    indexes: [
      "CREATE INDEX idx_device_event_device_id_occurred_at ON device_event (device_id, occurred_at)",
    ],
  });
  app.save(event);
}, (app) => {
  app.delete(app.findCollectionByNameOrId("device_event"));
  app.delete(app.findCollectionByNameOrId("device_reading"));
  app.delete(app.findCollectionByNameOrId("device"));
  app.delete(app.findCollectionByNameOrId("zone"));
});
