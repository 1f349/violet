select source,
       pre,
       destination,
       abs,
       cors,
       secure_mode,
       forward_host,
       forward_addr,
       ignore_cert
from routes
where active = true
