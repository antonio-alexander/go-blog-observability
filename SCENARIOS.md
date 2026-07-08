# Scenarios (github.com/antonio-alexander/go-blog-observability)

Although not impossible; it's fairly time consuming, both in terms of configuration and gathering the proper observability items, to put together scenarios to show certain sitautions. So this document will attempt to do just that with some in depth hypothesis and analysis.

## Using a [Redis] Cache to Reduce Query Latency By Avoiding the Database

As I said in the [README.md](./README.md):

> Lets say you have a database that only has five connections. If you have five concurrent requests, your latency will reflect the time it takes to process a request, make the database call and return it. If you have more than five concurrent requests, your latency will reflect the aforementioned processes, but because all the connections are being used, it also has to wait (i.e., queue) for a connection to become available. Although if the sql is optimized, this may be a short period of time, at scale it may be a flat amount of time added to __EVERY__ request

We know that we shuold improve the overall performance of a given read endpoint by introducing a cache and avoiding using the database which has fixed resources that signnificantly limit concurency. In this scenario, we create a situation where we have a fixed number of database connections, an empty cache and a client-side scenario where we perform concurrent requests for a set duration; with the number of concurrent requests exceeding the maximum number of databsae connections. Given that we know __A LOT__ about [caches](https://github.com/antonio-alexander/go-blog-cache), we can make plenty hypotheses.

A properly implemented cache will:

- will use few to no database connections once the cache is hydrated (e.g., read through caching)

> Once a cache is hydrated with data, attempts to read the same data should result in a cache hit and not involve the database

- will have a high cache hit/miss ratio

> All else equal (e.g. if the cache isn't invalidated for some reason), a hydrated cache should have a cache hit/miss ratio approaching 1

- average latency will remain flat

> after the initial blip hydrating the cache, average latency for the request should remain flat

Any cache will have to deal with the following edge cases:

- when a cache miss occurs
- when data is mutated
