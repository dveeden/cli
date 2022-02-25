# Tasks
* [x] Add CLI flags and map to query (group_by, filter, interval, granularity, etc.)
* [x] Add labels to output
* [ ] Add CSV output format
* [x] Add chart output format
  * [x] Image
    * https://github.com/wcharczuk/go-chart
    * https://github.com/gonum/plot
    * https://github.com/go-echarts/go-echarts
  * [ ] ASCII
    * https://github.com/gizak/termui
    * https://github.com/guptarohit/asciigraph
    * https://github.com/keithknott26/datadash
* [ ] Multiple metrics
* [ ] `--watch` auto-update functionality
* [ ] Use existing Kafka cluster in CLI context

# Examples

```shell
./confluent metrics query \
--metric io.confluent.kafka.server/request_bytes \
--kafka lkc-5o9mn \
--group-by metric.type \
--group-by metric.principal_id \
--interval 'PT5M/now|m' \
--granularity ALL
```

```shell
./confluent metrics query \
--metric io.confluent.kafka.server/received_records \
--kafka lkc-5o9mn \
--group-by metric.topic \
--group-limit 10 \
--interval 'PT5M/now|m' \
--granularity ALL
```

```shell
./confluent metrics query \
--output chart-html \
--metric io.confluent.kafka.server/received_records \
--kafka lkc-5o9mn \
--group-by metric.topic \
--group-limit 20 \
--interval 'P2D/now|m' \
--granularity PT15M \
 | browser
```
