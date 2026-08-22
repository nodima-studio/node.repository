# Base64 Column

Encodes a text or bytes column to Base64, or decodes Base64 text back to text.
Null values and all unselected columns are preserved.

## Configuration

- `column`: column to transform.
- `action`: `encode` or `decode`.
- `encoding`: padded `standard` Base64 or padded URL-safe `url` Base64.

Encoding accepts text and bytes. Decoding emits text and fails on a value that
is not valid for the selected Base64 encoding.
