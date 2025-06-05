import gzip
import json
import binascii

json_data = {
  "message": "Hello, Gzip World!",
  "status": "success",
  "data": {
    "items": [1, 2, 3, 4, 5],
    "description": "This is a test of gzip compression for cURL --data-raw."
  }
}
json_string = json.dumps(json_data)

import io
buf = io.BytesIO()
with gzip.GzipFile(fileobj=buf, mode='wb') as f:
    f.write(json_string.encode('utf-8'))

gzipped_data = buf.getvalue()

hex_escaped_data = ''.join([f'\\x{byte:02x}' for byte in gzipped_data])

print(f"Gzipped Data:\n{hex_escaped_data}")