.headers on
.mode column
.width 46 30 12 12 80

-- Lease spacing from runtime-event DB. These OP_LEASED events are emitted by
-- scheduler when the token bucket grants a lease, so they are the right DB
-- evidence for whether the queue limiter spaced HTTP starts.
with http_leases as (
  select
    json_extract(payload_json, '$.event.workflowId') as workflow_id,
    json_extract(payload_json, '$.event.opId') as op_id,
    json_extract(payload_json, '$.event.occurredAt') as leased_at,
    json_extract(payload_json, '$.event.queue') as queue_key
  from sessionstream_events
  where session_id = 'runtime:global'
    and json_extract(payload_json, '$.event.site') = 'hackernews'
    and json_extract(payload_json, '$.event.kind') = 'RUNTIME_EVENT_KIND_OP_LEASED'
    and json_extract(payload_json, '$.event.queue') = 'site:hackernews:http'
), spaced as (
  select
    workflow_id,
    op_id,
    leased_at,
    (julianday(leased_at) - julianday(lag(leased_at) over (partition by workflow_id order by leased_at))) * 86400.0 as seconds_since_previous_http_lease
  from http_leases
)
select
  workflow_id,
  leased_at,
  printf('%.3f', seconds_since_previous_http_lease) as seconds_since_prev_http_lease,
  case
    when seconds_since_previous_http_lease is null then 'first http lease in workflow'
    when seconds_since_previous_http_lease >= 1.0 then '>= 1s: policy spacing satisfied'
    else '< 1s: policy spacing violation'
  end as rate_limit_interpretation,
  op_id
from spaced
order by leased_at desc;
