# Design

## Summary

This receiver mimics what is provided by the Lightstep Microsatellites:

* Clock correction: The legacy tracers and this receiver send each other
 timestamp values in order to sync up any missmatched clocks.
 - `ReportRequest`: `TimestampOffsetMicros` includes an offset that MUST
   be applied to all timestamps being reported. This value is zero if
   no clock correction is required.
 - `ReportResponse`: This receiver sends two timestamps, `Receive` and
   `Transmit`, with the times at which the latest request was received
   and later answered with a response, in order to help the tracers
   adjust their offsets.

## TODO

* Implement gRPC support.
* Implement OBSReport.
* Consider mapping semantic conventions:
  - Values that can be consumed within the processor, e.g. detect event names from Logs, detect `StatusKind` from error tags.
  - Values that affect the entire OT ecosystem. Probably can be offered as a separate processor instead.
  - Lightstep-specific tags (attributes) that _may_ need to be mapped to become useful for OTel processors.
* `Baggage` is being sent as part of Lightstep's `SpanContext`, but it is not exported in any way at this point.
* Find all special Tags (e.g. "lightstep.component_name") and think which ones we should map
   (there is already a document about this somewhere).
* Legacy tracers send payloads using the `application/octet-stream` content type and using the
  `/api/v2/reports` path. We don't check for it but worth verifying this.
* Top level `ReporterId` is not being used at this moment.
