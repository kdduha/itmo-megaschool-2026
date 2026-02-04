# Benchmark

Here you can find a simple performance benchmark for the backend service + OpenAI. 
Configure base params in [main vars](./benchmark/main.go) and run the script with:
```sh
go run .
```

The script result would be a markdown table in the STDOUT (one row example is below)
```
| Format | Requests | Avg Time | Total Time |
|--------|----------|----------|------------|
| drawio | 2 | 54.889s | 1m49.777s |
```
