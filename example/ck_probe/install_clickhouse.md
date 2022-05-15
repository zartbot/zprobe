### Install clickhouse grafana

```bash
sudo apt-get install apt-transport-https ca-certificates dirmngr
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv E0C56BD4

echo "deb https://repo.clickhouse.tech/deb/stable/ main/" | sudo tee \
    /etc/apt/sources.list.d/clickhouse.list
sudo apt-get update

sudo apt-get install -y clickhouse-server clickhouse-client

sudo service clickhouse-server start


sudo apt install  adduser libfontconfig1 prometheus prometheus-node-exporter htop
wget https://dl.grafana.com/enterprise/release/grafana-enterprise_8.5.0_amd64.deb
sudo dpkg -i grafana-enterprise_8.5.0_amd64.deb
```

### Add plugin for grafana
```bash
grafana-cli plugins install vertamedia-clickhouse-datasource

sudo grafana-cli plugins install gowee-traceroutemap-panel
sudo grafana-cli plugins install novatec-sdg-panel
sudo grafana-cli plugins install agenty-flowcharting-panel

sudo service grafana-server stop
sudo service grafana-server start
```


grafana UI-ID for prometheuse node export

16098
11074

### Clickhouse SQL

```sql
select ASN,SPName,Dest,RespAddr,TTL,FlowKey,count(),avg(Delay) from zprobe WHERE Dest =='www.github.com' GROUP by Dest,RespAddr,TTL,FlowKey,ASN,SPName ORDER by TTL,FlowKey
```

### Clickhouse Sliding window(backup)


https://stackoverflow.com/questions/64733246/clickhouse-sliding-moving-window

Starting from version 21.4 added the full support of window-functions. At this moment it was marked as an experimental feature.


```sql
SELECT
    Time,
    groupArray(any(Value)) OVER (ORDER BY Time ASC ROWS BETWEEN 2 PRECEDING AND CURRENT ROW) AS Values
FROM 
(
    /* Emulate the test dataset, */
    select toDateTime(a) as Time, rowNumberInAllBlocks()+1 as Value
    from (
        select arrayJoin([
            '2020-01-01 12:11:00',
            '2020-01-01 12:12:00',
            '2020-01-01 12:13:00',
            '2020-01-01 12:14:00',
            '2020-01-01 12:15:00',
            '2020-01-01 12:16:00']) a
    )
    order by Time
)
GROUP BY Time
SETTINGS allow_experimental_window_functions = 1
```

/*
┌────────────────Time─┬─Values──┐
│ 2020-01-01 12:11:00 │ [1]     │
│ 2020-01-01 12:12:00 │ [1,2]   │
│ 2020-01-01 12:13:00 │ [1,2,3] │
│ 2020-01-01 12:14:00 │ [2,3,4] │
│ 2020-01-01 12:15:00 │ [3,4,5] │
│ 2020-01-01 12:16:00 │ [4,5,6] │
└─────────────────────┴─────────┘
*/
See https://altinity.com/blog/clickhouse-window-functions-current-state-of-the-art.

ClickHouse has several datablock-scoped window functions, let's take neighbor:

```sql
SELECT Time, [neighbor(Value, -2), neighbor(Value, -1), neighbor(Value, 0)] Values
FROM (
  /* emulate origin data */
  SELECT toDateTime(data.1) as Time, data.2 as Value
  FROM (
    SELECT arrayJoin([('2020-01-01 12:11:00', 1),
    ('2020-01-01 12:12:00', 2),
    ('2020-01-01 12:13:00', 3),
    ('2020-01-01 12:14:00', 4),
    ('2020-01-01 12:15:00', 5),
    ('2020-01-01 12:16:00', 6)]) as data)
  )
```
/*
┌────────────────Time─┬─Values──┐
│ 2020-01-01 12:11:00 │ [0,0,1] │
│ 2020-01-01 12:12:00 │ [0,1,2] │
│ 2020-01-01 12:13:00 │ [1,2,3] │
│ 2020-01-01 12:14:00 │ [2,3,4] │
│ 2020-01-01 12:15:00 │ [3,4,5] │
│ 2020-01-01 12:16:00 │ [4,5,6] │
└─────────────────────┴─────────┘

*/
An alternate way based on the duplication of source rows by window_size times:

```sql
SELECT   
  arrayReduce('max', arrayMap(x -> x.1, raw_result)) Time,
  arrayMap(x -> x.2, raw_result) Values
FROM (  
  SELECT groupArray((Time, Value)) raw_result, max(row_number) max_row_number
  FROM (
    SELECT 
      3 AS window_size,
      *, 
      rowNumberInAllBlocks() row_number,
      arrayJoin(arrayMap(x -> x + row_number, range(window_size))) window_id
    FROM (
      /* emulate origin dataset */
      SELECT toDateTime(data.1) as Time, data.2 as Value
      FROM (
        SELECT arrayJoin([('2020-01-01 12:11:00', 1),
          ('2020-01-01 12:12:00', 2),
          ('2020-01-01 12:13:00', 3),
          ('2020-01-01 12:14:00', 4),
          ('2020-01-01 12:15:00', 5),
          ('2020-01-01 12:16:00', 6)]) as data)
      ORDER BY Value
      )
    )
  GROUP BY window_id
  HAVING max_row_number = window_id
  ORDER BY window_id
  )
```  
/*
┌────────────────Time─┬─Values──┐
│ 2020-01-01 12:11:00 │ [1]     │
│ 2020-01-01 12:12:00 │ [1,2]   │
│ 2020-01-01 12:13:00 │ [1,2,3] │
│ 2020-01-01 12:14:00 │ [2,3,4] │
│ 2020-01-01 12:15:00 │ [3,4,5] │
│ 2020-01-01 12:16:00 │ [4,5,6] │
└─────────────────────┴─────────┘
*/
Extra example:

```sql
SELECT   
  arrayReduce('max', arrayMap(x -> x.1, raw_result)) id,
  arrayMap(x -> x.2, raw_result) values
FROM (  
  SELECT groupArray((id, value)) raw_result, max(row_number) max_row_number
  FROM (
    SELECT 
      48 AS window_size,
      *, 
      rowNumberInAllBlocks() row_number,
      arrayJoin(arrayMap(x -> x + row_number, range(window_size))) window_id
    FROM (
      /* the origin dataset */
      SELECT number AS id, number AS value
      FROM numbers(4096) 
      )
    )
  GROUP BY window_id
  HAVING max_row_number = window_id
  ORDER BY window_id
  )
  ```


┌─id─┬─values────────────────┐
│  0 │ [0]                   │
│  1 │ [0,1]                 │
│  2 │ [0,1,2]               │
│  3 │ [0,1,2,3]             │
│  4 │ [0,1,2,3,4]           │
│  5 │ [0,1,2,3,4,5]         │
│  6 │ [0,1,2,3,4,5,6]       │
│  7 │ [0,1,2,3,4,5,6,7]     │
..
│ 56 │ [9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56]  │
│ 57 │ [10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57] │
│ 58 │ [11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58] │
│ 59 │ [12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59] │
│ 60 │ [13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59,60] │
