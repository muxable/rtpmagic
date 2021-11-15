# demuxer

Demuxers split packet streams by a specific property.

The API for these deviates from the channel return pattern because it's too easy to accidentally block. Using a callback guarantees parallel processing.