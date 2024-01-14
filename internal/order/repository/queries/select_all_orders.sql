select
    id,
    number,
    user_login,
    uploaded_at,
    coalesce(accrual, 0),
    status
from gophermart."order"
where user_login = $1
order by uploaded_at desc;