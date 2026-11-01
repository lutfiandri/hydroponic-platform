# Hydroponic IoT Monitoring — Backend Specification

## Tech Stack

- **Backend:** PocketBase (SQLite)
- **Database Management:** PocketBase JavaScript Migrations
- **Architecture:** Relational database, no JSON fields
- **Telemetry Interval:** Every 10 minutes
- **Device Events:** Event-based (recorded when an event occurs)

## Overview

Build a backend for a hydroponic IoT monitoring system using PocketBase.

The system supports multiple zones, each containing multiple physical IoT devices (ESP32).

There are two device types:

- `tank` — A device installed at a hydroponic water reservoir.
- `table` — A device installed at the water inlet pipe of a hydroponic growing table.

Each device belongs to exactly one zone.

## 1. Database Collections

Create four PocketBase Base Collections using JavaScript migrations.

All collection names must be **singular**.

### Collection: `zone`

Represents a physical area, such as a greenhouse.

| Field         | Type | Constraints        |
| ------------- | ---- | ------------------ |
| `id`          | ID   | PocketBase default |
| `name`        | Text | Required           |
| `description` | Text | Optional           |

No parent or hierarchical relationship is required for now.

### Collection: `device`

Represents a physical ESP32 device.

| Field          | Type            | Constraints               |
| -------------- | --------------- | ------------------------- |
| `id`           | ID              | PocketBase default        |
| `zone_id`      | Relation → zone | Required, single          |
| `name`         | Text            | Required                  |
| `device_code`  | Text            | Required, unique          |
| `type`         | Select          | Required: `tank`, `table` |
| `status`       | Select          | `online`, `offline`       |
| `last_seen_at` | Date            | Optional                  |

Each device belongs to one zone.

### Collection: `device_reading`

Stores periodic sensor measurements.

Each device sends one reading approximately every 10 minutes.

| Field                | Type              | Constraints          |
| -------------------- | ----------------- | -------------------- |
| `id`                 | ID                | PocketBase default   |
| `device_id`          | Relation → device | Required, single     |
| `water_level`        | Number            | Optional, 0–100       |
| `has_water_level`    | Bool              | Default false        |
| `ppm`                | Number            | Optional, minimum 0  |
| `has_ppm`            | Bool              | Default false        |
| `water_temperature`  | Number            | Optional, Celsius    |
| `has_water_temperature` | Bool           | Default false        |
| `battery_percentage` | Number            | Optional, 0–100      |
| `has_battery_percentage` | Bool          | Default false        |
| `recorded_at`        | Date              | Required             |

Sensor fields are optional because different device types may report different measurements.
PocketBase Number fields are non-nullable and default to 0. Each sensor has a
`has_*` flag: true means a measurement exists, including a legitimate zero;
false means no measurement exists and the numeric value must be ignored.
Writers must explicitly send the matching flag with each measurement. These
flags are not inferred automatically. When clearing a measurement, set its
flag to false and its numeric value to 0.

Do not store historical readings as JSON.

Create an index on `(device_id, recorded_at)`.

### Collection: `device_event`

Stores hardware events, actuator activities, and automation execution results.

Events must be recorded when they occur, not at the 10-minute telemetry interval.

| Field         | Type              | Constraints                     |
| ------------- | ----------------- | ------------------------------- |
| `id`          | ID                | PocketBase default              |
| `device_id`   | Relation → device | Required, single                |
| `event_type`  | Text              | Required                        |
| `source`      | Select            | `automatic`, `manual`, `system` |
| `status`      | Select            | `pending`, `success`, `failed`  |
| `message`     | Text              | Optional                        |
| `occurred_at` | Date              | Required                        |

Example event types:

- `pump_on`
- `pump_off`
- `valve_open`
- `valve_close`
- `device_restarted`
- `device_error`

Use a text field for `event_type` to allow new events without database schema changes.

Create an index on `(device_id, occurred_at)`.

## 2. Entity Relationships

```text
zone
 └── device
      ├── device_reading
      └── device_event
```

- One zone has many devices.
- One device belongs to one zone.
- One device has many readings.
- One device has many events.

## 3. Example Data Structure

```text
Greenhouse A (zone)
├── Tank Controller (device: tank)
│   ├── device_reading
│   └── device_event
├── Table 1 Controller (device: table)
│   ├── device_reading
│   └── device_event
└── Table 2 Controller (device: table)
    ├── device_reading
    └── device_event

Greenhouse B (zone)
├── Tank Controller (device: tank)
└── Table 1 Controller (device: table)
```

## 4. Implementation Requirements

1. Use PocketBase JavaScript migrations to create collections and fields.
2. Define all relationships using PocketBase relation fields with an `_id` suffix. Relation values reference PocketBase record IDs; `device.device_code` is a separate external hardware identifier.
3. Use singular collection names exactly as specified.
4. Do not use JSON fields for telemetry or event records.
5. Add indexes for historical reading and event queries.
6. Use PocketBase default `id`, `created`, and `updated` fields.
7. Lock all five API access rules (`null`) on all four collections. Only PocketBase superusers may access them. The frontend uses an interactive superuser login; never embed superuser credentials in the frontend bundle.
8. Support querying latest readings and historical readings by device.
9. Support querying device events ordered by `occurred_at` descending.
10. Do not introduce unnecessary collections or abstractions.
11. Disable cascade deletion on all relations. Required relations prevent deleting zones or devices that still have dependent records.

## 5. Suggested Frontend Routes

```text
/dashboard

/zones
/zones/:zoneId

/devices
/devices/:deviceId
/devices/:deviceId/readings
/devices/:deviceId/events

/analytics
/activity
/settings
```

Frontend routes may use plural names for readability. The singular naming requirement applies to database collections.

## 6. Scope

For the MVP, implement only:

- Zone management
- Device management
- Periodic sensor readings
- Device and actuator event logging
- Historical telemetry queries
- Dashboard monitoring support

Do not implement complex automation rule engines, recursive zones, or additional device-channel collections yet.

Keep the architecture simple, maintainable, and extensible.
