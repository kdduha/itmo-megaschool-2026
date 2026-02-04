# Benchmark

Here you can find a simple performance benchmark for the backend service + OpenAI. 
Configure base params in [main vars](./benchmark/main.go) and run the script with:
```sh
go run .
```

The script result would be a markdown table in the STDOUT (one row example is below)
```md
...
2026/02/04 01:54:37 OK C1.png 27.245851375s
2026/02/04 01:55:28 OK Sequence.png 51.225438208s

## Benchmark Results

| Format | Requests | Avg Time | Total Time | Avg File Size |
|--------|----------|----------|------------|---------------|
| png | 3 | 44.279s | 2m12.836s | 98.83 KB |
| jpg | 3 | 45.236s | 2m15.707s | 252.25 KB |
...
```
