select
    id, number, user_login, uploaded_at, status, accrual
from gophermart.order
where user_login = $1
for update;