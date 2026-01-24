# Fix for go-trailblaze CDN Package

The issue is that aws-sdk-go-v2 adds `Accept-Encoding: gzip` header and includes it in the signature. Cloudflare modifies this header to `gzip, br` in transit, causing signature mismatch.

## Simple Fix - Add DisableAcceptEncodingGzip option

In go-trailblaze's `cdn.go`, add `DisableAcceptEncodingGzip: true` to the S3 client options:

```go
s3Client = s3.New(s3.Options{
    BaseEndpoint:              aws.String(endpoint),
    Region:                    region,
    Credentials:               creds,
    UsePathStyle:              true,
    HTTPClient:                httpClient,
    DisableAcceptEncodingGzip: true,  // <-- ADD THIS LINE - prevents Accept-Encoding from being signed
})
```

This tells the SDK to NOT add the `Accept-Encoding` header at all, so:

1. It won't be included in SignedHeaders
2. Cloudflare can modify it freely without breaking signatures

## Why this works

- AWS SDK v2 has built-in middleware that adds `Accept-Encoding: gzip` and signs it
- `DisableAcceptEncodingGzip: true` disables that middleware entirely
- The header won't be in the signature, so proxy modifications don't matter

## Note

The custom middleware approach (removing Accept-Encoding in Build step) doesn't work because the SDK's internal middleware adds the header AND includes it in the SignedHeaders list before any custom Build middleware runs.
