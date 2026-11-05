# Local SMTP/TLS test certificates

The integration fixture generates its CA, certificate, and private key in memory
at test runtime. This directory must not contain production credentials, fixed
private keys, or certificates. The fixture listens only on a loopback address
and never sends external email.
