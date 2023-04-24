select source,
       pre,
       destination,
       abs,
       code
from redirects
where active = true
