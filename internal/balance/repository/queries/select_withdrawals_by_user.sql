select
    id,
    order_number,
    user_login,
    sum,
    processed_at
from gophermart.withdrawals
where user_login = $1;