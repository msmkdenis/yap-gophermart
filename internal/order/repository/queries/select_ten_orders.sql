select
    id,
    number,
    user_login,
    uploaded_at,
    coalesce(accrual, 0),
    status
from gophermart."order"
where status not in ('INVALID', 'PROCESSED')
order by uploaded_at desc
limit 10;