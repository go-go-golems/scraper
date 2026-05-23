.headers on
.mode column
.width 46 10 26 26 80

-- Recent Hacker News workflows and their terminal status.
select
  id as workflow_id,
  status,
  created_at,
  updated_at,
  input_json
from workflows
where site = 'hackernews'
order by created_at desc
limit 20;

.print ''
.print 'Failed HTTP fetch ops for Hacker News, newest first'
.width 46 9 28 28 80 80
select
  o.workflow_id,
  o.status,
  o.created_at,
  r.completed_at,
  json_extract(o.input_json, '$.request.url') as request_url,
  json_extract(r.error_json, '$.Code') || ': ' || json_extract(r.error_json, '$.Message') as error
from ops o
left join results r on r.op_id = o.id
where o.site = 'hackernews'
  and o.kind = 'http/fetch'
  and o.status = 'failed'
order by o.created_at desc
limit 20;
