---
title: MIRACL English licensed sampling protocol
type: corpus-card
status: verified
created: 2026-10-12
updated: 2026-10-12
related: ["[[STUDY_PLAN]]", "[[RESULTS]]"]
---
Owner: Kyle Wild. Operational truth: https://github.com/dorkitude/embedding-local-cpu-vs-hosted/issues/1

License review completed before acquisition, 2026-10-12 UTC. MIRACL's official
repository and Hugging Face cards identify English as the 2019-02-01 Wikipedia
dump, 32,893,221 passages / 5,758,285 articles, segmented by WikiExtractor.
MIRACL licenses its contributions under Apache-2.0 (receipt: receipts/miracl-LICENSE).
This DOES NOT replace Wikipedia's underlying CC BY-SA 3.0 license applicable
to that 2019 dump. CC BY-SA permits copying, adaptation, commercial reuse and
redistribution, subject to attribution, license notice, indication of changes and
share-alike. This is broader than academic-only access.

We preserve original docid/title, article URL and history URL for each selected
passage, identify Wikipedia contributors as authors, retain source dump URL,
and license derived corpus/chunks under CC BY-SA 3.0. Wikimedia's Terms expressly
allow attribution by linking to the reused page (its history credits contributors).
The current terms refer to CC BY-SA 4.0; we conservatively preserve the 3.0 license
of the historical dump rather than treating today's terms as the original grant.
No images, tables, private documents or separately licensed media are acquired.
The recipe's transformations are passage selection, whitespace tokenization and
overlapping subdivision; original source text is otherwise retained locally.
Raw text is not committed. Attribution manifest is committed compressed, with
UTF-8 titles and properly escaped URLs. Republishing text requires keeping that
manifest, these notices, license text and change descriptions alongside it.

Primary receipts: official project README, pinned dataset cards, full Apache
and CC BY-SA legal text, Wikimedia Terms. URLs are captured in receipts/SOURCES.tsv.
Dataset pins: corpus d921ec7e349ce0d28daf30b2da9da5ee698bef0d;
queries/qrels 5be20db9509754dadad47689368639fcec739c00.

Sampling: retain all unique English dev judged documents (positive AND negative).
Scan every English corpus shard, select remaining documents by smallest SHA-256
of "embedding-study-20261012:" + docid. Freeze 100,000 documents before embedding.
Secondary 20,000 subset retains all judged documents and takes smallest hashes
from the primary distractors. All 799 native dev queries are used. This enriches
the sampled corpus with judged/hard documents and removes many full-corpus
distractors: scores are NOT official MIRACL full-corpus results. Unjudged documents
are treated as nonrelevant; false-negative relevance remains possible. Relevance
is document/passage level, never proof of an answer-bearing chunk.

Additional historical primary receipt captured during audit:
receipts/wikimedia-terms-2014-2023.html (official oldid=98524, linked by current
Wikimedia Terms). This is the Terms version applicable to the 2019 dump and
explicitly specifies CC BY-SA 3.0. It independently confirms the historical
license conclusion rather than relying on present-day 4.0 terms or HF metadata.
