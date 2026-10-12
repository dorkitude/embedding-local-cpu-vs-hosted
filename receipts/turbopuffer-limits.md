# Limits

There isn't a limit or performance metric we can't improve by an order of
magnitude when prioritized. If you expect to brush up against a limit or you
are limited by present performance, [contact us](/contact).

| Metric | Current limit |
| --- | --- |
| Max documents (global) | Unlimited (seen: 1T+ @ 3PB+)[^limits-note-1] |
| Max documents (per namespace) | 128B @ 256TB ( [seen: 100B @ 200TB](/blog/ann-v3) )[^limits-note-2] |
| Max [shards](/docs/sharding) per namespace | 256 |
| Max documents (per shard) | 500M @ 1TB[^limits-note-3] |
| Max number of namespaces | Unlimited (seen: 250M+)[^limits-note-4] |
| Max number of [pinned namespaces](/docs/pinning) | 256 |
| Max vector columns per namespace[^limits-note-5] | 8 |
| Maximum number of [embedded attributes per namespace](/docs/embedding) | 4 |
| Max dimensions for dense vectors | 10,752 |
| Max total dimensions for sparse vectors | Unlimited (seen: 30,522) |
| Max dimensions per sparse vector | 1,024 |
| Max inactive time in cache | hours[^limits-note-6] |
| Max write throughput (global) | Unlimited (seen: 10M+ writes/s @ 32GB/s)[^limits-note-7] |
| Max write throughput (per namespace) | 10k writes/s @ 32 MB/s[^limits-note-8] |
| Max namespace copy throughput[^limits-note-9] | 72 MB/s |
| Number of branches[^limits-note-10] | Unlimited (seen: 10M+) |
| Max upsert batch request size | 512 MB |
| Max upsert batch of [embedded attributes](/docs/embedding) | 256 rows |
| Max rows affected by [patch by filter](/docs/write#patch-by-filter) | 50k[^limits-note-11] |
| Max rows affected by [delete by filter](/docs/write#delete-by-filter) | 5M[^limits-note-12] |
| Max ingested, unindexed data[^limits-note-13] | 2 GB |
| Max queries (global) | Unlimited (seen: 100k+ queries/s)[^limits-note-14] |
| Max queries (per namespace)[^limits-note-15] | 5k+ queries/s |
| Max queries in a [multi-query request](/docs/query#param-queries) | 16 |
| Max concurrent queries per unpinned namespace[^limits-note-16] | 16 (100s of queries/s) |
| Max read replicas[^limits-note-17] | Unlimited (seen: 3) |
| Vector search recall@10[^limits-note-18] | 90-100% |
| Max attribute value size[^limits-note-19] | 8 MiB |
| Max filterable value size[^limits-note-20] | 4 KiB |
| Max document size | 64 MiB |
| Max id size | 64 bytes |
| Max attribute name length[^limits-note-21] | 128 bytes |
| Max attribute names per namespace | 1,024 |
| Max namespace name length[^limits-note-22] | 128 bytes |
| Max full-text query length | 8,192 |
| Max [limit.total](/docs/query#param-limit) | 10k |
| Max aggregation groups per query | 10k |
| Max [computed attributes](/docs/query#param-compute_attributes) per query | 256[^limits-note-23] |
| Max fragments per [highlight](/docs/fts#highlighting) | 1024[^limits-note-24] |

[^limits-note-1]: Per namespace limits still apply, but due to the extreme scalability of object storage, we don't see any problems storing trillions of documents across billions of namespaces.
[^limits-note-2]: Scaling beyond 500M documents @ 1TB requires a [sharded namespace](/docs/sharding) , which partitions data across up to 256 shards, each with those same limits.
[^limits-note-3]: Choose [num_shards](/docs/write#param-sharding) so that each shard stays under the byte size limit. The document count limit is soft - we recommend not exceeding it for optimal performance, but performance ultimately depends on query complexity. Shard count cannot be changed in place, so size for growth. If you need a different count later, [copy into a new namespace](/docs/sharding#configuration) with a new num_shards.
[^limits-note-4]: Each namespace is simply a prefix on object storage, which means it scales to virtually unlimited as object storage providers don't specify a limit. [Architecture](/docs/architecture) for more details
[^limits-note-5]: The number of vector columns is fixed at namespace creation time and cannot be changed. Eventually up to 128. Vector columns for embedded attributes count toward this limit.
[^limits-note-6]: This is simply the current production average cache SSD expiry time. We are currently over-provisioned in cache capacity. In the future, this may change slightly as we optimize the economies of scale. However, consider that our incentive is aligned with yours to keep the cache intelligent, as using object storage for hot reads is expensive.
[^limits-note-7]: Writes are processed through a cluster of horizontally scaleable turbopuffer Rust binaries. They write directly to object storage, which scales to virtually unlimited writes. Per namespace limits still apply. Read more on the [architecture page](/docs/architecture)
[^limits-note-8]: Limited by indexing performance, see "Max ingested, unindexed data". We are constantly working on improving indexing performance.
[^limits-note-9]: Throughput for [copy_from_namespace](/docs/write#param-copy_from_namespace). Cross-region copies may be 20-35% slower depending on distance. See [Cross-Region Backups](/docs/backups) for details.
[^limits-note-10]: There are no limits on branching. This means there is no limit on how many children branches a namespace can have (A->B, A->C, A->D,...) nor on the length of chains of branches (A->B, B->C, C->D,...). See the [branching guide](/docs/branching) for details.
[^limits-note-11]: Your write will contain a `rows_remaining` field indicating whether any rows were skipped. You can issue a duplicate patch_by_filter request to patch more rows. This limit is there to ensure that indexing and consistent queries can keep up with patches.
[^limits-note-12]: Your write will contain a `rows_remaining` field indicating whether any rows were skipped. You can issue a duplicate delete_by_filter request to delete more rows. This limit is there to ensure that indexing and consistent queries can keep up with deletes.
[^limits-note-13]: Ingested data is asynchronously indexed (see [architecture docs](/docs/architecture) for details). It is possible to ingest faster than we can index, causing a backlog. If the indexing backlog reaches this limit, upsert requests will return HTTP 429 until the backlog decreases. This ensures queries can be executed without excessive resource use and latency. We continuously improve indexing throughput. You can see the number of unindexed documents by sending a query and examining `.performance.exhaustive_search_count` in the response.
[^limits-note-14]: Due to turbopuffer's simple, horizontally scalable architecture with a cluster of Rust binaries pointing to object storage, scaling reads is easy. Per namespace limits still apply. Read more about batching on the [architecture page](/docs/architecture)
[^limits-note-15]: Adding read replicas can raise this limit to arbitrarily high values (contact us). Replicas will be added automatically in future versions of turbopuffer.
[^limits-note-16]: If this is exceeded, the 17th query waits up to 800ms to start. If it can't claim the semaphore in that window, it will return an HTTP 429. This limit serves to mitigate the noisy neighbour effect, and can be raised by contacting us or using [namespace pinning](/docs/pinning) to reserve capacity. Note that query latency interacts with this limit to determine effective max QPS. For 50ms queries, the default limit allows >300 QPS. Eventually consistent queries can be enabled to improve throughput even further. As we improve performance, effective QPS will increase. Aggregate / group-by queries scan the namespace and use more CPU and I/O than typical vector or BM25 queries, so each one consumes 4 slots from this limit instead of 1. Exact-distance (kNN) vector queries are also more CPU and I/O intensive than ANN and consume 2 slots. Adding read replicas can raise this limit to arbitrarily high values (contact us). Replicas will be added automatically in future versions of turbopuffer.
[^limits-note-17]: Adding read replicas can raise concurrent queries and QPS to arbitrarily high values (contact us). Replicas will be automatically added in future versions of turbopuffer. For pinned namespaces, see [Pinned Replicas](/docs/pinning#replicas) for details.
[^limits-note-18]: This means the top 10 returned from turbopuffer has on average 90-95% of the true top 10. In the future, this is tunable, but currently the configuration is hardcoded for this recall. Vector search is fundamentally about the tradeoff between recall and other attributes like cost and performance. You can read more on our blog about how we monitor recall.
[^limits-note-19]: How large any individual attribute value can be. For arrays, this represents the total byte size of the array; there is no limit on the number of elements in an array.
[^limits-note-20]: How large any filterable attribute value can be. For arrays, this represents the maximum size of any element; there is no limit on the number of elements in an array.
[^limits-note-21]: Attribute names cannot start with $
[^limits-note-22]: Must match `[A-Za-z0-9-_.]{1,128}`
[^limits-note-23]: Computed attributes are evaluated for every result a query returns; this bounds the work a single query can do. Contact us if you need more.
[^limits-note-24]: Highlighting splits text into fragments and returns the best-matching ones. This caps fragment_limit, the number returned per result. Contact us if you need more.


---

This page: [/docs/limits.md](https://turbopuffer.com/docs/limits.md)

All documentation pages: [/llms.txt](https://turbopuffer.com/llms.txt)

All documentation in one file: [/llms-full.txt](https://turbopuffer.com/llms-full.txt)
