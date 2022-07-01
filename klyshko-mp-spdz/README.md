# Klyshko Fake Correlated Randomness Generator

Provides a Klyshko *Correlated Randomness Generator* (CRG) that uses the MP-SPDZ
ability to generate fake tuples.

For a high-level description of the Klyshko subsystem, its components, and how
these interact, please see the [README] at the root of this repository.

## Foreign MAC Key Shares

The MP-SPDZ fake tuple generator requires all MAC key shares to be available to
all parties. They are expected to be made available using the
[KII mechanism for providing extra parameters][kii-extra] using a config map
like the following:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: io.carbynestack.engine.params.extra
data:
  1_mac_key_share_p: |
    ...
  1_mac_key_share_2: |
   ...
  2_mac_key_share_p: |
    ...
  2_mac_key_share_2: |
    ...
```

There must be entries for all VCPs except the local one (although it doesn't
hurt if that is also provided). The numeric prefix of the key specifies the
number (zero-based) of the VCP this key is for.

## KII Tuple Type Mapping

The mapping from [KII] tuple types to the flags required for invoking the
`Fake-Offline.x` executable is as follows:

| KII Tuple Type             | Flag           | Folder                | Header Length |
| -------------------------- | -------------- | --------------------- | ------------- |
| BIT_GFP                    | --nbits 0,n    | 2-p-128/Bits-p-P0     | 37            |
| BIT_GF2N                   | --nbits n,0    | 2-2-40/Bits-2-P0      | 34            |
| INPUT_MASK_GFP             | --ntriples 0,n | 2-p-128/Triples-p-P0  | 37            |
| INPUT_MASK_GF2N            | --ntriples n,0 | 2-2-40/Triples-2-P0   | 34            |
| INVERSE_TUPLE_GFP          | --ninverses n  | 2-p-128/Inverses-p-P0 | 37            |
| INVERSE_TUPLE_GF2N         | --ninverses n  | 2-2-40/Inverses-2-P0  | 34            |
| SQUARE_TUPLE_GFP           | --nsquares 0,n | 2-p-128/Squares-p-P0  | 37            |
| SQUARE_TUPLE_GF2N          | --nsquares n,0 | 2-2-40/Squares-2-P0   | 34            |
| MULTIPLICATION_TRIPLE_GFP  | --ntriples 0,n | 2-p-128/Triples-p-P0  | 37            |
| MULTIPLICATION_TRIPLE_GF2N | --ntriples n,0 | 2-2-40/Triples-2-P0   | 34            |

[kii]: ../README.md#klyshko-integration-interface-kii
[kii-extra]: ../README.md#additional-parameters
[readme]: ../README.md
