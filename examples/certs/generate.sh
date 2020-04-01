# Generate root CA
cfssl gencert -initca ca.json | cfssljson -bare ca

# Generate server cert
cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=config.json   -profile=server server.json | cfssljson -bare server

# Generate client cert + key
cfssl gencert -ca ca.pem -ca-key ca-key.pem csr.json | cfssljson -bare client
