.headers on
.mode column
.width 14 28 14 30

-- Current queue limiter state. The table stores token bucket state per site/queue.
select
  site,
  queue_key,
  tokens,
  last_refill_at
from queue_limit_state
order by site, queue_key;

.print ''
.print 'Hacker News ops grouped by queue and status'
.width 28 16 10 8 30 30
select
  queue_key,
  kind,
  status,
  count(*) as count,
  min(created_at) as first_created_at,
  max(updated_at) as last_updated_at
from ops
where site = 'hackernews'
group by queue_key, kind, status
order by queue_key, kind, status;

.print ''
.print 'Raw site workflow inputs for failed Hacker News runs'
.width 46 10 80
select id as workflow_id, status, input_json
from workflows
where site = 'hackernews'
order by created_at desc
limit 10;
