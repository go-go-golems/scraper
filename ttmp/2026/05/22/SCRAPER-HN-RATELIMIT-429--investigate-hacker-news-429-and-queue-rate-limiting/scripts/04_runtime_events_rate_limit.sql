.headers on
.mode column
.width 9 30 36 46 80

-- Runtime-event backend events from sessionstream for Hacker News scheduler / rate-limit behavior.
-- This DB stores sessionstream payload JSON; paths below use protobuf JSON field names.
select
  ordinal,
  json_extract(payload_json, '$.event.occurredAt') as occurred_at,
  json_extract(payload_json, '$.event.kind') as kind,
  json_extract(payload_json, '$.event.opId') as op_id,
  json_extract(payload_json, '$.event.message') as message
from sessionstream_events
where session_id = 'runtime:global'
  and json_extract(payload_json, '$.event.site') = 'hackernews'
  and json_extract(payload_json, '$.event.kind') in (
    'RUNTIME_EVENT_KIND_OP_LEASED',
    'RUNTIME_EVENT_KIND_OP_SUCCEEDED',
    'RUNTIME_EVENT_KIND_OP_FAILED',
    'RUNTIME_EVENT_KIND_QUEUE_RATE_LIMITED',
    'RUNTIME_EVENT_KIND_WORKFLOW_UPDATED'
  )
order by ordinal desc
limit 120;

.print ''
.print 'Rate-limit events only'
.width 9 30 46 80
select
  ordinal,
  json_extract(payload_json, '$.event.occurredAt') as occurred_at,
  json_extract(payload_json, '$.event.queue') as queue,
  payload_json
from sessionstream_events
where session_id = 'runtime:global'
  and json_extract(payload_json, '$.event.site') = 'hackernews'
  and json_extract(payload_json, '$.event.kind') = 'RUNTIME_EVENT_KIND_QUEUE_RATE_LIMITED'
order by ordinal desc
limit 80;
