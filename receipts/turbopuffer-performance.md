# Optimizing Performance

**Upsert** (Time for the batch to be durably acknowledged by object storage. Documents are immediately available to consistent reads after this.)
- Upsert latency (512kb docs): p50=165ms, p90=248ms, p99=850ms

**Vector Query** (768 dimensions, f16, 10M docs, ~15GB. Strongly consistent.)
- warm (10M docs): p50=14ms, p90=17ms, p99=27ms
- cold (10M docs): p50=874ms, p90=1214ms, p99=1686ms

**Full-Text Search** (BM25, 1M docs, ~300MB. Strongly consistent.)
- warm (1M docs): p50=13ms, p90=18ms, p99=29ms
- cold (1M docs): p50=316ms, p90=381ms, p99=559ms

turbopuffer is designed to be performant by default, but there are ways to
optimize performance further. These suggestions aren't requirements for good
performance—rather, they highlight opportunities for improvement when you have
the flexibility to choose.

For example, while a single namespace with 100M documents works fine, splitting
it into 10 namespaces of 10M documents each may yield better query performance
if there's a natural way to group the documents.

These suggestions apply generally. For specific strategies to optimize ingestion throughput, read our [Ingestion guide](/docs/ingestion).

## Choose the Closest Region

Choose the [region][region] closest to your backend. We can't beat the
speed of light. If there isn't a region close to us and the latency is
paramount, [contact us.](/contact)

## HTTP Connection Reuse

Use the same `Turbopuffer` client instance for as
many requests as possible. This uses a connection pool behind the scenes to
avoid the overhead of a TCP and TLS handshake on every request.

## Use U64 or UUID IDs

The smaller the IDs, the faster the puffin'. A UUID
encoded as a string is 36 bytes, whereas the [UUID-native type is 16
bytes][schema]. A u64 is even smaller at 8 bytes.

## Inverted Index

Attribute values that are filterable are indexed into an inverted
index. Inverted indexes means large intersects can be much faster than on a
traditional B-Tree index.

## Disable Filtering for Unfiltered Attributes

For attributes you never intend to filter on, marking
[attributes as `filterable: false`][schema] will improve indexing performance and
grant you a 50% discount. For large attribute values, e.g. storing a raw text
chunk or image, this can improve performance and cost significantly.

## Use Small Namespaces

The rule of thumb is to make the namespaces as
small as they can be without having to routinely query more than one at a time.
If documents have significantly different schemas, it's also worth splitting
them. Don't try to be too clever. Smaller namespaces will be faster to query and index.

## Shard Large Namespaces That Can't Be Split

Some documents must stay in the same namespace, to be searched together in a
single query. If such a namespace approaches the [size limit of a single
index](/docs/limits), or its size starts bottlenecking indexing performance,
[namespace sharding](/docs/sharding) transparently partitions it across multiple
internal shards, each indexing its slice independently.

## Prewarm Namespaces

If your application is latency-sensitive, consider
[warming the cache][warm] for the namespace before the user interacts with it
(e.g. when they open the search or chat dialog).

## Use Smaller Vectors

Smaller vectors will be faster to search, e.g.
512 dimensions will be faster than 1536 dimensions.
[f16](/docs/write#param-type) will be faster than f32, and
[i8](/docs/write#param-type) faster still. The tradeoff with
smaller vectors is typically lower search precision. Consider the cost/performance
vs precision tradeoff with your own evals. For models with quantization-aware
training ([voyage-4 series][voyage-4], [voyage-context-3][voyage-context-3],
[embed-v4][embed-v4], [Qwen3-VL-Embedding-8B][qwen3-vl]), `int8` output
matches `f32` precision ([benchmarks][voyage-benchmarks]).
Create the namespace with an [`[N]i8`](/docs/write#param-type) vector type and pass the `int8` values directly as JSON integers (in the range `-128` to `127`) .

## Use Branching to Duplicate Namespaces

If you're creating copies of namespaces for testing, backups, or code repositories,
[branching](/docs/branching) creates a copy-on-write clone in constant time regardless of
namespace size.

## Batch Writes

If you're writing a lot of documents, consider batching
them into fewer writes. This will improve performance and [leverages batch
discounts up to 50%][pricing]. Each individual write batch request can be a
maximum of 512MB.

## Concurrent Writes

If you're writing a lot of documents, consider using
multiple processes to write batches in parallel. Especially for single-threaded
runtimes like Node.js or Python, this can be a significant performance boost as
upserting is generally bottlenecked by serialization and compression.

## Control include_attributes

The more data we have to
return, the slower it will be. Make sure to only specify the attributes you
need.

## Use Eventual Consistency

If you need higher query throughput and can tolerate
stale results, consider using [eventual consistency](/docs/query#param-consistency)
for your queries.

## Pin High QPS Namespaces

For sustained high query volumes over a few, large
namespaces, consider [namespace pinning](/docs/pinning) to provision reserved
compute nodes for more predictable cost and performance with always-warm cache.

## Avoid Large Attributes with Frequent Patches

When using [patch][patch]
or [patch_by_filter][patch_by_filter], turbopuffer currently reads all
attributes of the documents being patched, even those not being modified. If you
have large attributes (>10KB), consider storing them in a separate namespace
linked by id. For example, if you have chunks with vectors
and a large metadata blob that's shared across chunks, store the metadata in a
separate namespace keyed by a shared id (e.g. `file_id`). At query time, do a
vector search on chunks, then look up the metadata using the unique ids from
your results. This way, patches to chunk-specific attributes never touch the
large metadata.

## Keep First-Stage Ranking Simple

`rank_by` expressions can quickly become quite sophisticated. For best
performance, we recommend keeping the first-stage ranking function simple, with
only a few attributes being used to compute BM25 scores and/or attribute
scores, retrieving in the order of 100 to 1,000 hits, and then applying more
sophisticated ranking in the second stage.
* **Understand how Glob and Regex filters are optimized.** Under the hood, they
use a trigram-based index to quickly narrow down the set of possibly matching candidates.
As a general rule of thumb, the more specific the pattern, the better the performance. Anchored
patterns (`turbo*` or `*puffer`) are much more specific than unanchored patterns (`*tpuf*`),
and thus will perform better. Avoid unspecific patterns like `[a-z]*`, which require a full-table
scan.
* **Separate ANN and BM25 index namespaces**. If indexing performance suffers on a
namespace with both ANN and BM25 indexes, we recommend splitting these indexes into
separate namespaces to improve throughput. We're actively working on improving
performance for combined ANN and BM25 namespaces, and this temporary workaround will
be unnecessary soon.

[patch]: /docs/write#param-patch_rows
[patch_by_filter]: /docs/write#param-patch_by_filter
[warm]: /docs/warm-cache
[schema]: /docs/write#schema
[region]: /docs/regions
[pricing]: /#pricing
[voyage-benchmarks]: https://docs.google.com/spreadsheets/d/1qLBWWvN7-4W53BveJgkQiDSoK_j2RYLh5DafDdEOPnc/edit?gid=1834510862#gid=1834510862&range=A11:B14
[voyage-4]: https://blog.voyageai.com/2026/01/15/voyage-4/
[voyage-context-3]: https://blog.voyageai.com/2025/07/23/voyage-context-3/
[embed-v4]: https://cohere.com/blog/embed-4
[qwen3-vl]: https://qwen.ai/blog?id=qwen3-vl-embedding


---

This page: [/docs/performance.md](https://turbopuffer.com/docs/performance.md)

All documentation pages: [/llms.txt](https://turbopuffer.com/llms.txt)

All documentation in one file: [/llms-full.txt](https://turbopuffer.com/llms-full.txt)
