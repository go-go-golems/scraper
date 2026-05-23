.headers on
.mode column
.width 46 12 46 80

-- Evidence that the 429s line up with malformed Hacker News pagination URLs.
select
  o.workflow_id,
  o.status,
  o.id as op_id,
  json_extract(o.input_json, '$.request.url') as request_url,
  json_extract(r.error_json, '$.Details.response.statusCode') as status_code,
  json_extract(r.error_json, '$.Details.response.finalURL') as final_url
from ops o
left join results r on r.op_id = o.id
where o.site = 'hackernews'
  and o.kind = 'http/fetch'
  and json_extract(o.input_json, '$.request.url') like '%?p=2/?p=3%'
order by o.created_at desc;

.print ''
.print 'Count by request URL and outcome'
.width 80 10 8
select
  json_extract(o.input_json, '$.request.url') as request_url,
  o.status,
  count(*) as count
from ops o
where o.site = 'hackernews'
  and o.kind = 'http/fetch'
group by request_url, o.status
order by request_url, o.status;
