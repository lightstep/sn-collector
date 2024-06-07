# Design

## Summary

This receiver exposes *very* basic functionality, with only http/protobuf support
(initially). Details are:

* `ReportRequest` is the protobuf we send/receive, with `ReportRequest.Report`
  being similar to `Resource` (e.g. `Resource` has attributes in its `Tags` attribute).
* Legacy tracers send the service name as `lightstep.component_name` in
  `ReportRequest.Report.Tags`, and we derive the actual OTel `service.name`
  from it, falling back to `unknown_service`.
* We do a **raw** ingestion/conversion, meaning we don't do any semconv mapping,
 other than deriving `service.name` from `lightstep.component_name`. See
 TODO below.
* Legacy tracers send 64 bits TraceIds, which we convert to 128 bytes OTel ids.
* Clock correction: Some legacy tracers (Java) perform clock correction, sending
 along a timeoffset to be applied, and expecting back Receive/Transmit
 timestamps from the microsatellites/collector:
 - `ReportRequest`: `TimestampOffsetMicros` includes an offset that MUST
   be applied to all timestamps being reported. This value is zero if
   no clock correction is required.
 - `ReportResponse`: This receiver sends two timestamps, `Receive` and
   `Transmit`, with the times at which the latest request was received
   and later answered with a response, in order to help the tracers
   adjust their offsets.

## TODO

* Implement OBSReport.
* Legacy tracers send payloads using the `application/octet-stream` content type and using the
  `/api/v2/reports` path. We don't check for it but worth verifying this (at least the
  content-type).
* Top level `ReporterId` is not being used at this moment.
* `Baggage` is being sent as part of Lightstep's `SpanContext`, but it is not exported in any way at this point.
* Find all special Tags (e.g. "lightstep.*") and think which ones we should map.
* Implement gRPC support.
* Implement Thrift support.
* Consider mapping semantic conventions:
  - Values that can be consumed within the processor, e.g. detect event names from Logs, detect `StatusKind` from error tags.
    - Consider using the OpenTracing compatibilty section in the Specification, which states how to process errors and multiple parents.
  - Values that affect the entire OT ecosystem. Probably can be offered as a separate processor instead.
  - Lightstep-specific tags (attributes) that _may_ need to be mapped to become useful for OTel processors.
