.headers on
.mode column

.schema leases

.print ''
.print 'Recent Hacker News lease rows'
.width 46 46 20 26 26
select *
from leases
where op_id in (select id from ops where site = 'hackernews')
order by rowid desc
limit 50;
