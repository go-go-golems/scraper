.headers on
.mode column
.width 46 9 30 26 26 10 80

-- Timing between Hacker News http/fetch leases/completions. This answers
-- whether scraper spaced HTTP fetches by queue policy or started them back-to-back.
with fetches as (
  select
    o.workflow_id,
    o.id as op_id,
    o.status,
    o.created_at,
    r.completed_at,
    json_extract(o.input_json, '$.request.url') as request_url,
    json_extract(r.error_json, '$.Code') as error_code,
    lag(r.completed_at) over (order by o.created_at) as previous_fetch_completed_at,
    lag(json_extract(o.input_json, '$.request.url')) over (order by o.created_at) as previous_request_url,
    (julianday(o.created_at) - julianday(lag(r.completed_at) over (order by o.created_at))) * 86400.0 as seconds_since_previous_fetch_completed,
    (julianday(o.created_at) - julianday(lag(o.created_at) over (order by o.created_at))) * 86400.0 as seconds_since_previous_fetch_created
  from ops o
  left join results r on r.op_id = o.id
  where o.site = 'hackernews'
    and o.kind = 'http/fetch'
)
select
  workflow_id,
  status,
  created_at,
  completed_at,
  printf('%.3f', seconds_since_previous_fetch_completed) as sec_after_prev_done,
  printf('%.3f', seconds_since_previous_fetch_created) as sec_after_prev_created,
  request_url,
  error_code
from fetches
order by created_at desc
limit 30;
