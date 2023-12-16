select
    id,
    number,
    user_login,
    uploaded_at,
    status
from gophermart."order"
where user_login = $1
order by uploaded_at desc;