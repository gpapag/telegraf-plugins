# Ps Input Plugin

The `ps` plugin executes the `ps` command on Linux machines 
periodically and parses metrics from their output in any one 
of the accepted [Input Data Formats](https://github.com/influxdata/telegraf/blob/master/docs/DATA_FORMATS_INPUT.md).

### Configuration:

```toml
[[inputs.ps]]
  ## Timeout for each command to complete.
  timeout = "5s"
```
